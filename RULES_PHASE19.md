# Phase 19: Inventory & Bulk System

## Objective

Implement a proper inventory system with Bulk tracking, nested containers, and automatic Encumbered/Immobilised condition application when carrying capacity is exceeded.

---

## 1. Inventory Structure

**Target File:** `pkg/rules/entity/inventory.go`

```go
package entity

import "uaa/vdnd/pkg/rules/item"

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
    Items       []InventoryItem
    CoinsCP     int // Copper pieces
    CoinsSP     int // Silver pieces
    CoinsGP     int // Gold pieces
    CoinsPP     int // Platinum pieces
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
            sum += child.TotalBulk()
            sum += calculateContentsBulk(child.ItemID)
        }

        if parent, ok := itemMap[parentID]; ok {
            // checks if item is *item.GeneralItem (or *item.Backpack) and returns BulkReduction value. Must handle type assertions safely.
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

    totalCoins := inv.CoinsCP + inv.CoinsSP + inv.CoinsGP + inv.CoinsPP
    coinBulk := BulkValue((totalCoins / 1000) * 10)
    return total + coinBulk
}
```

---

## 2. Add Bulk to Weapons

(Standard Weapon Bulk logic remains as per Phase 19 specification)

---

## 3. General Item Type

(Standard General Item types remain as per Phase 19 specification, ensuring `GetBulkReduction()` is exposed)

---

## 4. Entity Integration

(Entity BulkLimit, MaxBulk, and CurrentBulk logic remains, ensuring `CurrentBulk()` uses `Inventory.TotalBulk()`)

---

## 5. Encumbrance Condition Updates

(Standard Speed penalty and Clumsy logic remains)

---

## 6. Automatic Encumbrance Application

(PickUpItem and DropItem convenience wrappers remain)

---

## 7. Tests

Adjusted to test nested containers:

```go
func TestContainerBulkReduction(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)
    ent.Abilities.Strength = 10 // limit 5

    // Add Backpack (0 Bulk, 2 Bulk reduction)
    backpack := item.StandardItems["backpack"]
    ent.PickUpItem(backpack.ID, backpack.Name, entity.BulkValue(backpack.Bulk*10), 1, backpack)

    // Add 3 Bulk of items
    ent.PickUpItem("item1", "Heavy Item", entity.Bulk1, 3, nil)

    // Move items into backpack
    ent.Inventory.MoveItem("item1", "backpack")
    ent.UpdateEncumbranceConditions()

    // Total Bulk should be 1 (3 - 2 reduction)
    if ent.CurrentBulkInWholeBulk() != 1 {
        t.Errorf("Expected 1 Bulk, got %d", ent.CurrentBulkInWholeBulk())
    }
}
```

---

## 8. CLI Commands

Updated command set:

```bash
# Move item into container
vd inventory move <item_id> --to <container_id>

# View inventory (should show nested items)
vd entity get paladin --field inventory
```