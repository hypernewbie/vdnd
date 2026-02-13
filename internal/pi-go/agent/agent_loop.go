package agent

import (
	"context"
	"fmt"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

// AgentLoop starts an agent loop with new prompt messages.
// The prompts are added to the context and events are emitted for them.
// Returns an event stream that can be iterated for agent events.
func AgentLoop(
	ctx context.Context,
	prompts []Message,
	agentCtx AgentContext,
	config LoopConfig,
) *AgentEventStream {
	stream := NewAgentEventStream()

	go func() {
		// Copy messages and append prompts
		messages := make([]Message, len(agentCtx.Messages), len(agentCtx.Messages)+len(prompts))
		copy(messages, agentCtx.Messages)
		messages = append(messages, prompts...)

		currentCtx := AgentContext{
			SystemPrompt: agentCtx.SystemPrompt,
			Messages:     messages,
			Tools:        agentCtx.Tools,
		}

		newMessages := make([]Message, len(prompts))
		copy(newMessages, prompts)

		stream.Push(Event{Type: EventAgentStart})
		stream.Push(Event{Type: EventTurnStart})
		for _, prompt := range prompts {
			p := prompt
			stream.Push(Event{Type: EventMessageStart, Message: &p})
			stream.Push(Event{Type: EventMessageEnd, Message: &p})
		}

		runLoop(ctx, &currentCtx, newMessages, config, stream)
	}()

	return stream
}

// AgentLoopContinue continues an agent loop from the current context without adding a new message.
// Used for retries — context already has user message or tool results.
func AgentLoopContinue(
	ctx context.Context,
	agentCtx AgentContext,
	config LoopConfig,
) (*AgentEventStream, error) {
	if len(agentCtx.Messages) == 0 {
		return nil, fmt.Errorf("cannot continue: no messages in context")
	}

	last := agentCtx.Messages[len(agentCtx.Messages)-1]
	if last.Role == ai.RoleAssistant {
		return nil, fmt.Errorf("cannot continue from message role: assistant")
	}

	stream := NewAgentEventStream()

	go func() {
		currentCtx := AgentContext{
			SystemPrompt: agentCtx.SystemPrompt,
			Messages:     append([]Message(nil), agentCtx.Messages...),
			Tools:        agentCtx.Tools,
		}

		stream.Push(Event{Type: EventAgentStart})
		stream.Push(Event{Type: EventTurnStart})

		runLoop(ctx, &currentCtx, nil, config, stream)
	}()

	return stream, nil
}

// runLoop is the main loop logic shared by AgentLoop and AgentLoopContinue.
func runLoop(
	ctx context.Context,
	agentCtx *AgentContext,
	newMessages []Message,
	config LoopConfig,
	stream *AgentEventStream,
) {
	firstTurn := true

	// Check for steering messages at start
	var pendingMessages []Message
	if config.GetSteeringMessages != nil {
		msgs, err := config.GetSteeringMessages()
		if err == nil {
			pendingMessages = msgs
		}
	}

	// Outer loop: continues when queued follow-up messages arrive after agent would stop
	for {
		hasMoreToolCalls := true
		var steeringAfterTools []Message

		// Inner loop: process tool calls and steering messages
		for hasMoreToolCalls || len(pendingMessages) > 0 {
			if !firstTurn {
				stream.Push(Event{Type: EventTurnStart})
			} else {
				firstTurn = false
			}

			// Process pending messages (inject before next assistant response)
			if len(pendingMessages) > 0 {
				for _, msg := range pendingMessages {
					m := msg
					stream.Push(Event{Type: EventMessageStart, Message: &m})
					stream.Push(Event{Type: EventMessageEnd, Message: &m})
					agentCtx.Messages = append(agentCtx.Messages, m)
					newMessages = append(newMessages, m)
				}
				pendingMessages = nil
			}

			// Stream assistant response
			assistantMsg, err := streamAssistantResponse(ctx, agentCtx, config, stream)
			if err != nil {
				// Create error message
				errMsg := makeErrorMessage(config.Model, err.Error(), ai.StopReasonError)
				newMessages = append(newMessages, ai.NewAssistantMsg(errMsg))
				stream.Push(Event{Type: EventTurnEnd, Message: &Message{Role: ai.RoleAssistant, Assistant: &errMsg}, ToolResults: nil})
				stream.Push(Event{Type: EventAgentEnd, Messages: newMessages})
				stream.End(newMessages)
				return
			}

			assistantMessage := ai.NewAssistantMsg(assistantMsg)
			newMessages = append(newMessages, assistantMessage)

			if assistantMsg.StopReason == ai.StopReasonError || assistantMsg.StopReason == ai.StopReasonAborted {
				stream.Push(Event{Type: EventTurnEnd, Message: &assistantMessage, ToolResults: nil})
				stream.Push(Event{Type: EventAgentEnd, Messages: newMessages})
				stream.End(newMessages)
				return
			}

			// Check for tool calls
			var toolCalls []ai.ContentBlock
			for _, c := range assistantMsg.Content {
				if c.Type == ai.ContentTypeToolCall {
					toolCalls = append(toolCalls, c)
				}
			}
			hasMoreToolCalls = len(toolCalls) > 0

			var toolResults []ai.ToolResultMessage
			if hasMoreToolCalls {
				results, steering := executeToolCalls(ctx, agentCtx.Tools, assistantMsg, stream, config.GetSteeringMessages)
				toolResults = results
				steeringAfterTools = steering

				for _, result := range toolResults {
					resultMsg := ai.NewToolResultMsg(result)
					agentCtx.Messages = append(agentCtx.Messages, resultMsg)
					newMessages = append(newMessages, resultMsg)
				}
			}

			stream.Push(Event{Type: EventTurnEnd, Message: &assistantMessage, ToolResults: toolResults})

			// Get steering messages after turn completes
			if len(steeringAfterTools) > 0 {
				pendingMessages = steeringAfterTools
				steeringAfterTools = nil
			} else if config.GetSteeringMessages != nil {
				msgs, err := config.GetSteeringMessages()
				if err == nil {
					pendingMessages = msgs
				}
			}
		}

		// Agent would stop here. Check for follow-up messages.
		if config.GetFollowUpMessages != nil {
			followUps, err := config.GetFollowUpMessages()
			if err == nil && len(followUps) > 0 {
				pendingMessages = followUps
				continue
			}
		}

		break
	}

	stream.Push(Event{Type: EventAgentEnd, Messages: newMessages})
	stream.End(newMessages)
}

// streamAssistantResponse streams an LLM response, transforming agent messages to LLM messages.
func streamAssistantResponse(
	ctx context.Context,
	agentCtx *AgentContext,
	config LoopConfig,
	stream *AgentEventStream,
) (ai.AssistantMessage, error) {
	// Apply context transform if configured
	messages := agentCtx.Messages
	if config.TransformContext != nil {
		transformed, err := config.TransformContext(ctx, messages)
		if err != nil {
			return ai.AssistantMessage{}, fmt.Errorf("context transform failed: %w", err)
		}
		messages = transformed
	}

	// Convert to LLM-compatible messages
	llmMessages, err := config.ConvertToLlm(messages)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("convertToLlm failed: %w", err)
	}

	// Build LLM context
	llmCtx := ai.Context{
		SystemPrompt: agentCtx.SystemPrompt,
		Messages:     llmMessages,
	}
	// Convert agent tools to ai.Tool for the LLM
	if len(agentCtx.Tools) > 0 {
		aiTools := make([]ai.Tool, len(agentCtx.Tools))
		for i, t := range agentCtx.Tools {
			aiTools[i] = t.Tool
		}
		llmCtx.Tools = aiTools
	}

	// Resolve API key
	apiKey := config.APIKey
	if config.GetApiKey != nil {
		resolved, err := config.GetApiKey(config.Model.Provider)
		if err == nil && resolved != "" {
			apiKey = resolved
		}
	}

	// Stream
	opts := &ai.SimpleStreamOptions{
		StreamOptions: ai.StreamOptions{
			APIKey:    apiKey,
			SessionID: config.SessionID,
		},
		Reasoning:       config.Reasoning,
		ThinkingBudgets: config.ThinkingBudgets,
	}
	if config.MaxRetryDelayMs != nil {
		opts.MaxRetryDelayMs = config.MaxRetryDelayMs
	}

	llmStream, err := ai.StreamSimple(ctx, config.Model, llmCtx, opts)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("stream failed: %w", err)
	}

	var partialMessage *ai.AssistantMessage
	addedPartial := false

	for event := range llmStream.Events() {
		switch event.Type {
		case ai.EventStart:
			partialMessage = event.Partial
			agentCtx.Messages = append(agentCtx.Messages, ai.NewAssistantMsg(*partialMessage))
			addedPartial = true
			m := ai.NewAssistantMsg(*partialMessage)
			stream.Push(Event{Type: EventMessageStart, Message: &m})

		case ai.EventTextStart, ai.EventTextDelta, ai.EventTextEnd,
			ai.EventThinkingStart, ai.EventThinkingDelta, ai.EventThinkingEnd,
			ai.EventToolCallStart, ai.EventToolCallDelta, ai.EventToolCallEnd:
			if partialMessage != nil {
				partialMessage = event.Partial
				agentCtx.Messages[len(agentCtx.Messages)-1] = ai.NewAssistantMsg(*partialMessage)
				m := ai.NewAssistantMsg(*partialMessage)
				e := event
				stream.Push(Event{
					Type:                  EventMessageUpdate,
					Message:               &m,
					AssistantMessageEvent: &e,
				})
			}

		case ai.EventDone, ai.EventError:
			finalMessage, err := llmStream.Result(ctx)
			if err != nil {
				return ai.AssistantMessage{}, err
			}
			if addedPartial {
				agentCtx.Messages[len(agentCtx.Messages)-1] = ai.NewAssistantMsg(finalMessage)
			} else {
				agentCtx.Messages = append(agentCtx.Messages, ai.NewAssistantMsg(finalMessage))
			}
			if !addedPartial {
				m := ai.NewAssistantMsg(finalMessage)
				stream.Push(Event{Type: EventMessageStart, Message: &m})
			}
			fm := ai.NewAssistantMsg(finalMessage)
			stream.Push(Event{Type: EventMessageEnd, Message: &fm})
			return finalMessage, nil
		}
	}

	// Fallback: get result
	result, err := llmStream.Result(ctx)
	if err != nil {
		return ai.AssistantMessage{}, err
	}
	return result, nil
}

// executeToolCalls executes tool calls from an assistant message sequentially.
// Returns tool results and any steering messages that interrupted execution.
func executeToolCalls(
	ctx context.Context,
	tools []Tool,
	assistantMsg ai.AssistantMessage,
	stream *AgentEventStream,
	getSteeringMessages GetMessagesFunc,
) ([]ai.ToolResultMessage, []Message) {
	var toolCalls []ai.ContentBlock
	for _, c := range assistantMsg.Content {
		if c.Type == ai.ContentTypeToolCall {
			toolCalls = append(toolCalls, c)
		}
	}

	var results []ai.ToolResultMessage
	var steeringMessages []Message

	for i, tc := range toolCalls {
		// Find the tool
		var tool *Tool
		for j := range tools {
			if tools[j].Name == tc.Name {
				tool = &tools[j]
				break
			}
		}

		stream.Push(Event{
			Type:       EventToolExecutionStart,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Args:       tc.Arguments,
		})

		var toolResult ToolResult
		var isError bool

		if tool == nil {
			toolResult = ToolResult{
				Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: fmt.Sprintf("Tool %q not found", tc.Name)}},
			}
			isError = true
		} else {
			// Validate arguments
			validatedArgs, err := ai.ValidateToolArguments(tool.Tool, ai.ToolCall{
				Type: ai.ContentTypeToolCall, ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments,
			})
			if err != nil {
				toolResult = ToolResult{
					Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: err.Error()}},
				}
				isError = true
			} else {
				// Execute
				result, err := tool.Execute(ctx, tc.ID, validatedArgs, func(partial ToolResult) {
					stream.Push(Event{
						Type:          EventToolExecutionUpdate,
						ToolCallID:    tc.ID,
						ToolName:      tc.Name,
						Args:          tc.Arguments,
						PartialResult: &partial,
					})
				})
				if err != nil {
					toolResult = ToolResult{
						Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: err.Error()}},
					}
					isError = true
				} else {
					toolResult = result
				}
			}
		}

		stream.Push(Event{
			Type:       EventToolExecutionEnd,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Result:     &toolResult,
			IsError:    isError,
		})

		toolResultMsg := ai.ToolResultMessage{
			Role:       ai.RoleToolResult,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Content:    toolResult.Content,
			Details:    toolResult.Details,
			IsError:    isError,
			Timestamp:  time.Now().UnixMilli(),
		}

		results = append(results, toolResultMsg)
		rm := ai.NewToolResultMsg(toolResultMsg)
		stream.Push(Event{Type: EventMessageStart, Message: &rm})
		stream.Push(Event{Type: EventMessageEnd, Message: &rm})

		// Check for steering messages — skip remaining tools if user interrupted
		if getSteeringMessages != nil {
			steering, err := getSteeringMessages()
			if err == nil && len(steering) > 0 {
				steeringMessages = steering
				// Skip remaining tool calls
				for _, skipped := range toolCalls[i+1:] {
					results = append(results, skipToolCall(skipped, stream))
				}
				break
			}
		}
	}

	return results, steeringMessages
}

// skipToolCall creates a skipped tool result for a tool call that was not executed.
func skipToolCall(tc ai.ContentBlock, stream *AgentEventStream) ai.ToolResultMessage {
	result := ToolResult{
		Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "Skipped due to queued user message."}},
	}

	stream.Push(Event{
		Type:       EventToolExecutionStart,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Args:       tc.Arguments,
	})
	stream.Push(Event{
		Type:       EventToolExecutionEnd,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Result:     &result,
		IsError:    true,
	})

	toolResultMsg := ai.ToolResultMessage{
		Role:       ai.RoleToolResult,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Content:    result.Content,
		IsError:    true,
		Timestamp:  time.Now().UnixMilli(),
	}

	rm := ai.NewToolResultMsg(toolResultMsg)
	stream.Push(Event{Type: EventMessageStart, Message: &rm})
	stream.Push(Event{Type: EventMessageEnd, Message: &rm})

	return toolResultMsg
}

// makeErrorMessage creates an error AssistantMessage.
func makeErrorMessage(model ai.Model, errMsg string, reason ai.StopReason) ai.AssistantMessage {
	return ai.AssistantMessage{
		Role:         ai.RoleAssistant,
		Content:      []ai.ContentBlock{{Type: ai.ContentTypeText, Text: ""}},
		Api:          model.Api,
		Provider:     model.Provider,
		Model:        model.ID,
		Usage:        ai.Usage{},
		StopReason:   reason,
		ErrorMessage: errMsg,
		Timestamp:    time.Now().UnixMilli(),
	}
}
