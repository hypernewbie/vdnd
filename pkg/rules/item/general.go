package item

import "uaa/vdnd/pkg/rules/trait"

type ItemCategory string

const (
	CategoryAdventuringGear ItemCategory = "adventuring_gear"
	CategoryConsumable      ItemCategory = "consumable"
	CategoryTool            ItemCategory = "tool"
	CategoryMaterial        ItemCategory = "material"
	CategoryContainer       ItemCategory = "container"
)

type GeneralItem struct {
	ID          string
	Name        string
	Category    ItemCategory
	Bulk        int // Whole bulk, 0 = Light, -1 = Negligible
	Price       int // In copper pieces
	Description string
	Traits      trait.TraitSet

	// For containers
	BulkCapacity  int // How much bulk it can hold (0 = not a container)
	BulkReduction int // How much bulk is reduced when items inside (e.g., backpack)
}

func NewGeneralItem(id, name string, category ItemCategory, bulk, price int) *GeneralItem {
	return &GeneralItem{
		ID:       id,
		Name:     name,
		Category: category,
		Bulk:     bulk,
		Price:    price,
	}
}

func (g *GeneralItem) GetBulkReduction() int {
	return g.BulkReduction
}

// Common adventuring gear
var StandardItems = map[string]*GeneralItem{
	"backpack":      {ID: "backpack", Name: "Backpack", Category: CategoryContainer, Bulk: 0, BulkCapacity: 40, BulkReduction: 20}, // 4 Bulk capacity, reduces 2 Bulk
	"bedroll":       {ID: "bedroll", Name: "Bedroll", Category: CategoryAdventuringGear, Bulk: 0, Price: 10},                       // Light
	"rope_50ft":     {ID: "rope_50ft", Name: "Rope (50 ft)", Category: CategoryAdventuringGear, Bulk: 1, Price: 100},
	"torch":         {ID: "torch", Name: "Torch", Category: CategoryAdventuringGear, Bulk: 0, Price: 1},
	"rations_week":  {ID: "rations_week", Name: "Rations (1 week)", Category: CategoryConsumable, Bulk: 0, Price: 40}, // Light
	"waterskin":     {ID: "waterskin", Name: "Waterskin", Category: CategoryAdventuringGear, Bulk: 0, Price: 5},
	"thieves_tools": {ID: "thieves_tools", Name: "Thieves' Tools", Category: CategoryTool, Bulk: 0, Price: 300},
	"healer_kit":    {ID: "healer_kit", Name: "Healer's Tools", Category: CategoryTool, Bulk: 1, Price: 500},
}
