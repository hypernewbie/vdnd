package entity

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
)

// BulkValue represents bulk as an integer where 10 = 1 Bulk (to handle Light items)
// Light items = 1, Bulk 1 = 10, Bulk 2 = 20, etc.
// Negligible = 0
type BulkValue int

const (
	BulkNegligible BulkValue = 0
	BulkLight      BulkValue = 1
	Bulk1          BulkValue = 10
	Bulk2          BulkValue = 20
	Bulk3          BulkValue = 30
)

// ToBulk converts internal value to display Bulk (rounds down, 10 Light = 1 Bulk)
func (b BulkValue) ToBulk() int {
	return int(b) / 10
}

// ToLight returns the Light remainder
func (b BulkValue) LightRemainder() int {
	return int(b) % 10
}

// String returns human-readable bulk like "2 Bulk, 3 Light" or "L" for light-only
func (b BulkValue) String() string {
	bulk := b.ToBulk()
	light := b.LightRemainder()
	if bulk == 0 && light == 0 {
		return "-"
	}
	if bulk == 0 {
		return fmt.Sprintf("%dL", light)
	}
	if light == 0 {
		return fmt.Sprintf("%d", bulk)
	}
	return fmt.Sprintf("%d, %dL", bulk, light)
}

type InventoryItem struct {
	Item     interface{} // *item.Weapon, *item.Armor, *item.Shield, or *item.GeneralItem
	ItemID   string
	ParentID string // If empty, carried directly. If set, ItemID of container.
	Name     string
	Quantity int
	Bulk     BulkValue // Per-item bulk
}

func (i InventoryItem) TotalBulk() BulkValue {
	return BulkValue(int(i.Bulk) * i.Quantity)
}

type Inventory struct {
	Items   []InventoryItem
	CoinsCP int // Copper pieces
	CoinsSP int // Silver pieces
	CoinsGP int // Gold pieces
	CoinsPP int // Platinum pieces
}

func NewInventory() *Inventory {
	return &Inventory{
		Items: make([]InventoryItem, 0),
	}
}

// TotalBulk calculates total bulk of all items plus coins, accounting for containers
func (inv *Inventory) TotalBulk() BulkValue {
	// Map of ParentID -> Children
	children := make(map[string][]InventoryItem)
	var roots []InventoryItem

	// Map to look up items by ID (needed for BulkReduction logic)
	// Using a pointer to GeneralItem to access BulkReduction
	itemMap := make(map[string]InventoryItem)

	for _, item := range inv.Items {
		itemMap[item.ItemID] = item
		if item.ParentID == "" {
			roots = append(roots, item)
		} else {
			children[item.ParentID] = append(children[item.ParentID], item)
		}
	}

	// Helper to recursively calculate bulk of contents
	var calculateContentsBulk func(string) BulkValue
	calculateContentsBulk = func(parentID string) BulkValue {
		sum := BulkValue(0)
		for _, child := range children[parentID] {
			// Base bulk of the item
			sum += child.TotalBulk()
			// Recursive bulk of its own contents
			sum += calculateContentsBulk(child.ItemID)
		}

		// Apply reduction if the parent is a container (like a backpack)
		// We only look at GeneralItem for now as they have BulkReduction
		// We need an interface or type switch to find GeneralItem
		if parent, ok := itemMap[parentID]; ok {
			// This is a bit hacky due to interface{}, but matches the registry structure
			// We'll use a helper to check for reduction
			reduction := getBulkReduction(parent.Item)
			if reduction > 0 {
				redVal := BulkValue(reduction)
				if sum > redVal {
					sum -= redVal
				} else {
					sum = 0
				}
			}
		}

		return sum
	}

	total := BulkValue(0)
	for _, root := range roots {
		total += root.TotalBulk()
		total += calculateContentsBulk(root.ItemID)
	}

	// 1000 coins = 1 Bulk
	totalCoins := inv.CoinsCP + inv.CoinsSP + inv.CoinsGP + inv.CoinsPP
	coinBulk := BulkValue((totalCoins / 1000) * 10)
	return total + coinBulk
}

// getBulkReduction extracts BulkReduction if the item is a GeneralItem
func getBulkReduction(itemRef interface{}) int {
	type reducer interface {
		GetBulkReduction() int
	}
	if r, ok := itemRef.(reducer); ok {
		return r.GetBulkReduction()
	}
	return 0
}

// AddItem adds an item to inventory, stacking if already present
func (inv *Inventory) AddItem(id, name string, bulk BulkValue, quantity int, itemRef interface{}) {
	for i := range inv.Items {
		if inv.Items[i].ItemID == id {
			inv.Items[i].Quantity += quantity
			return
		}
	}
	inv.Items = append(inv.Items, InventoryItem{
		ItemID:   id,
		Name:     name,
		Bulk:     bulk,
		Quantity: quantity,
		Item:     itemRef,
	})
}

// MoveItem puts an item inside a container
func (inv *Inventory) MoveItem(itemID, parentID string) error {
	found := false
	for i := range inv.Items {
		if inv.Items[i].ItemID == itemID {
			inv.Items[i].ParentID = parentID
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("item %s not found", itemID)
	}
	return nil
}

// RemoveItem removes quantity of an item, returns true if successful
func (inv *Inventory) RemoveItem(id string, quantity int) bool {
	for i := range inv.Items {
		if inv.Items[i].ItemID == id {
			if inv.Items[i].Quantity < quantity {
				return false
			}
			inv.Items[i].Quantity -= quantity
			if inv.Items[i].Quantity == 0 {
				inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			}
			return true
		}
	}
	return false
}

// GetItem returns the inventory entry for an item ID, or nil
func (inv *Inventory) GetItem(id string) *InventoryItem {
	for i := range inv.Items {
		if inv.Items[i].ItemID == id {
			return &inv.Items[i]
		}
	}
	return nil
}

// AddCoins adds coins of specified type
func (inv *Inventory) AddCoins(cp, sp, gp, pp int) {
	inv.CoinsCP += cp
	inv.CoinsSP += sp
	inv.CoinsGP += gp
	inv.CoinsPP += pp
}

// TotalWealthInCP returns total wealth converted to copper
func (inv *Inventory) TotalWealthInCP() int {
	return inv.CoinsCP + (inv.CoinsSP * 10) + (inv.CoinsGP * 100) + (inv.CoinsPP * 1000)
}

// --- Bulk Calculation Methods for Entity ---

// BulkLimit returns maximum bulk before Encumbered (5 + STR mod)
func (e *Entity) BulkLimit() int {
	strMod := e.Abilities.Modifier(ability.Strength)
	limit := 5 + strMod
	if limit < 0 {
		limit = 0
	}
	return limit
}

// MaxBulk returns maximum bulk before immobilised (10 + STR mod)
func (e *Entity) MaxBulk() int {
	strMod := e.Abilities.Modifier(ability.Strength)
	max := 10 + strMod
	if max < 0 {
		max = 0
	}
	return max
}

// CurrentBulk calculates total bulk being carried
func (e *Entity) CurrentBulk() BulkValue {
	total := BulkValue(0)

	// Inventory items
	if e.Inventory != nil {
		total += e.Inventory.TotalBulk()
	}

	// Worn armour (worn armour is typically 1 less bulk while worn)
	if e.WornArmor != nil {
		// Worn armour reduces bulk by 1 (min 0)
		armorBulk := e.WornArmor.Bulk - 1
		if armorBulk < 0 {
			armorBulk = 0
		}
		total += BulkValue(armorBulk * 10)
	}

	// Worn shield
	if e.WornShield != nil {
		total += BulkValue(e.WornShield.Bulk * 10)
	}

	// Wielded weapons
	for _, w := range e.WieldedWeapons {
		if w.Bulk == 0 {
			total += BulkLight // Light weapons
		} else {
			total += BulkValue(w.Bulk * 10)
		}
	}

	return total
}

// CurrentBulkInWholeBulk returns bulk as whole number (for limit comparison)
func (e *Entity) CurrentBulkInWholeBulk() int {
	return e.CurrentBulk().ToBulk()
}

// IsEncumbered returns true if over bulk limit
func (e *Entity) IsEncumbered() bool {
	return e.CurrentBulkInWholeBulk() > e.BulkLimit()
}

// IsOverMaxBulk returns true if over max bulk (immobilised)
func (e *Entity) IsOverMaxBulk() bool {
	return e.CurrentBulkInWholeBulk() > e.MaxBulk()
}

// --- Automatic Encumbrance Application ---

// UpdateEncumbranceConditions checks bulk and applies/removes conditions
func (e *Entity) UpdateEncumbranceConditions() {
	currentBulk := e.CurrentBulkInWholeBulk()
	limit := e.BulkLimit()
	max := e.MaxBulk()

	// Over max = Immobilized
	if currentBulk > max {
		if !e.Conditions.Has(condition.Immobilized) {
			e.Conditions.Apply(condition.NewCondition(condition.Immobilized, "Over maximum bulk"))
		}
		// Also encumbered
		if !e.Conditions.Has(condition.Encumbered) {
			e.Conditions.Apply(condition.NewCondition(condition.Encumbered, "Over bulk limit"))
		}
		return
	}

	// Remove immobilized if we're under max
	e.Conditions.Remove(condition.Immobilized)

	// Over limit = Encumbered
	if currentBulk > limit {
		if !e.Conditions.Has(condition.Encumbered) {
			e.Conditions.Apply(condition.NewCondition(condition.Encumbered, "Over bulk limit"))
		}
		return
	}

	// Under limit = remove encumbered
	e.Conditions.Remove(condition.Encumbered)
}

// Convenience wrappers that auto-update conditions
func (e *Entity) PickUpItem(id, name string, bulk BulkValue, quantity int, itemRef interface{}) {
	e.Inventory.AddItem(id, name, bulk, quantity, itemRef)
	e.UpdateEncumbranceConditions()
}

func (e *Entity) DropItem(id string, quantity int) bool {
	success := e.Inventory.RemoveItem(id, quantity)
	if success {
		e.UpdateEncumbranceConditions()
	}
	return success
}
