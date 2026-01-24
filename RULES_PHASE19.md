# Phase 19: Inventory & Bulk System

## Objective

Implement a proper inventory system with Bulk tracking and automatic Encumbered/Immobilised condition application when carrying capacity is exceeded.

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

// TotalBulk calculates total bulk of all items plus coins
func (inv *Inventory) TotalBulk() BulkValue {
    total := BulkValue(0)
    for _, item := range inv.Items {
        total += item.TotalBulk()
    }
    // 1000 coins = 1 Bulk
    totalCoins := inv.CoinsCP + inv.CoinsSP + inv.CoinsGP + inv.CoinsPP
    coinBulk := BulkValue((totalCoins / 1000) * 10)
    return total + coinBulk
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
```

---

## 2. Add Bulk to Weapons

**Target File:** `pkg/rules/item/weapon.go`

Add `Bulk` field to `Weapon` struct:

```go
type Weapon struct {
    ID             string
    Name           string
    Category       WeaponCategory
    Group          WeaponGroup
    Damage         dice.DieRoll
    DamageType     DamageType
    Hands          int
    Traits         trait.TraitSet
    RangeIncrement int
    Bulk           int // Add this field (in whole Bulk units, 0 = Light)

    // ... existing fields
}
```

Update `NewWeapon` to accept bulk:

```go
func NewWeapon(id, name string, cat WeaponCategory, group WeaponGroup,
    damage dice.DieRoll, damageType DamageType, hands int,
    rangeIncrement int, bulk int, traits ...trait.TraitID) Weapon {

    w := Weapon{
        // ... existing assignments
        Bulk: bulk,
    }
    // ... rest of function
    return w
}
```

Update standard weapons in registry with correct bulk values:

```go
var StandardWeapons = map[string]Weapon{
    "longsword":  NewWeapon("longsword", "Longsword", CatMartial, GroupSword, dice.D8, DamageSlashing, 1, 0, 1, trait.TraitVersatile),
    "shortsword": NewWeapon("shortsword", "Shortsword", CatMartial, GroupSword, dice.D6, DamagePiercing, 1, 0, 1, trait.TraitAgile, trait.TraitFinesse),
    "dagger":     NewWeapon("dagger", "Dagger", CatSimple, GroupKnife, dice.D4, DamagePiercing, 1, 10, 0, trait.TraitAgile, trait.TraitFinesse, trait.TraitThrown), // Light = 0
    "greatsword": NewWeapon("greatsword", "Greatsword", CatMartial, GroupSword, dice.D12, DamageSlashing, 2, 0, 2, trait.TraitVersatile),
    "longbow":    NewWeapon("longbow", "Longbow", CatMartial, GroupBow, dice.D8, DamagePiercing, 2, 100, 2, trait.TraitDeadly, trait.TraitVolley),
}
```

---

## 3. General Item Type

**Target File:** `pkg/rules/item/general.go`

For non-weapon, non-armour items (adventuring gear, consumables, etc.):

```go
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
    BulkCapacity int  // How much bulk it can hold (0 = not a container)
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

// Common adventuring gear
var StandardItems = map[string]*GeneralItem{
    "backpack":      {ID: "backpack", Name: "Backpack", Category: CategoryContainer, Bulk: 0, BulkCapacity: 40, BulkReduction: 20}, // 4 Bulk capacity, reduces 2 Bulk
    "bedroll":       {ID: "bedroll", Name: "Bedroll", Category: CategoryAdventuringGear, Bulk: 0, Price: 10},  // Light
    "rope_50ft":     {ID: "rope_50ft", Name: "Rope (50 ft)", Category: CategoryAdventuringGear, Bulk: 1, Price: 100},
    "torch":         {ID: "torch", Name: "Torch", Category: CategoryAdventuringGear, Bulk: 0, Price: 1},
    "rations_week":  {ID: "rations_week", Name: "Rations (1 week)", Category: CategoryConsumable, Bulk: 0, Price: 40}, // Light
    "waterskin":     {ID: "waterskin", Name: "Waterskin", Category: CategoryAdventuringGear, Bulk: 0, Price: 5},
    "thieves_tools": {ID: "thieves_tools", Name: "Thieves' Tools", Category: CategoryTool, Bulk: 0, Price: 300},
    "healer_kit":    {ID: "healer_kit", Name: "Healer's Tools", Category: CategoryTool, Bulk: 1, Price: 500},
}
```

---

## 4. Entity Integration

**Target File:** `pkg/rules/entity/entity.go`

### 4.1 Add Inventory Field

Add to `Entity` struct:

```go
Inventory *Inventory
```

Update `NewEntity()`:

```go
func NewEntity(id, name string, level int) *Entity {
    return &Entity{
        // ... existing fields
        Inventory: NewInventory(),
    }
}
```

### 4.2 Bulk Calculation Methods

**Target File:** `pkg/rules/entity/inventory.go` (add to entity package)

```go
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
```

### 4.3 Update Clone() for Inventory

```go
// In Clone() method
if e.Inventory != nil {
    clone.Inventory = &Inventory{
        Items:   make([]InventoryItem, len(e.Inventory.Items)),
        CoinsCP: e.Inventory.CoinsCP,
        CoinsSP: e.Inventory.CoinsSP,
        CoinsGP: e.Inventory.CoinsGP,
        CoinsPP: e.Inventory.CoinsPP,
    }
    copy(clone.Inventory.Items, e.Inventory.Items)
}
```

---

## 5. Encumbrance Condition Updates

**Target File:** `pkg/rules/condition/effects.go`

Encumbered condition applies Clumsy 1 and -10 ft speed penalty.

```go
// GetSpeedPenalty returns speed penalty from conditions
func (t *ConditionTracker) GetSpeedPenalty() int {
    penalty := 0

    if t.Has(Encumbered) {
        penalty -= 10
    }

    if t.Has(Slowed) {
        // Slowed doesn't directly reduce speed, but affects actions
    }

    // Immobilized = speed 0
    if t.Has(Immobilized) {
        return -9999 // Effectively 0 speed
    }

    return penalty
}

// GetEncumbranceClumsy returns clumsy value from encumbrance
func (t *ConditionTracker) GetEncumbranceClumsy() int {
    if t.Has(Encumbered) {
        return 1
    }
    return 0
}
```

**Target File:** `pkg/rules/entity/combat.go`

Update speed calculation:

```go
func (e *Entity) GetSpeed() int {
    speed := e.BaseSpeed

    // Armour speed penalty (if STR too low)
    if e.WornArmor != nil {
        speed += e.WornArmor.EffectiveSpeedPenalty(e.Abilities.Get(ability.Strength))
    }

    // Shield speed penalty (tower shield)
    if e.WornShield != nil {
        speed += e.WornShield.SpeedPenalty
    }

    // Condition penalties
    speed += e.Conditions.GetSpeedPenalty()

    if speed < 0 {
        speed = 0
    }
    return speed
}
```

---

## 6. Automatic Encumbrance Application

**Target File:** `pkg/rules/entity/inventory.go`

Call this after any inventory change:

```go
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
```

---

## 7. Tests

**Target File:** `pkg/rules/entity/inventory_test.go`

```go
package entity_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/condition"
    "uaa/vdnd/pkg/rules/entity"
)

func TestBulkCalculation(t *testing.T) {
    inv := entity.NewInventory()

    // Add items
    inv.AddItem("rope", "Rope", entity.Bulk1, 2, nil)  // 2 Bulk
    inv.AddItem("torch", "Torch", entity.BulkLight, 5, nil) // 5 Light

    total := inv.TotalBulk()
    // 2 Bulk = 20, 5 Light = 5, total = 25 = 2 Bulk, 5 Light
    if total.ToBulk() != 2 {
        t.Errorf("Expected 2 Bulk, got %d", total.ToBulk())
    }
    if total.LightRemainder() != 5 {
        t.Errorf("Expected 5 Light remainder, got %d", total.LightRemainder())
    }
}

func TestCoinBulk(t *testing.T) {
    inv := entity.NewInventory()
    inv.AddCoins(500, 300, 200, 0) // 1000 coins = 1 Bulk

    total := inv.TotalBulk()
    if total.ToBulk() != 1 {
        t.Errorf("1000 coins should be 1 Bulk, got %d", total.ToBulk())
    }
}

func TestBulkLimits(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)
    ent.Abilities.Set(ability.Strength, 14) // +2 mod

    // Bulk limit = 5 + 2 = 7
    if ent.BulkLimit() != 7 {
        t.Errorf("Expected bulk limit 7, got %d", ent.BulkLimit())
    }

    // Max bulk = 10 + 2 = 12
    if ent.MaxBulk() != 12 {
        t.Errorf("Expected max bulk 12, got %d", ent.MaxBulk())
    }
}

func TestEncumbranceAutoApply(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)
    ent.Abilities.Set(ability.Strength, 10) // +0 mod
    // Bulk limit = 5, Max = 10

    // Add 6 Bulk worth of items (over limit)
    for i := 0; i < 6; i++ {
        ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
    }

    if !ent.Conditions.Has(condition.Encumbered) {
        t.Error("Should be Encumbered at 6 Bulk (limit 5)")
    }

    // Drop 2 items (now 4 Bulk)
    ent.DropItem("item_0", 1)
    ent.DropItem("item_1", 1)

    if ent.Conditions.Has(condition.Encumbered) {
        t.Error("Should not be Encumbered at 4 Bulk (limit 5)")
    }
}

func TestImmobilizedAtMaxBulk(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)
    ent.Abilities.Set(ability.Strength, 10) // +0 mod
    // Max = 10

    // Add 11 Bulk worth of items
    for i := 0; i < 11; i++ {
        ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
    }

    if !ent.Conditions.Has(condition.Immobilized) {
        t.Error("Should be Immobilized at 11 Bulk (max 10)")
    }
    if !ent.Conditions.Has(condition.Encumbered) {
        t.Error("Should also be Encumbered")
    }
}

func TestEncumberedSpeedPenalty(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)
    ent.BaseSpeed = 25
    ent.Abilities.Set(ability.Strength, 10)

    normalSpeed := ent.GetSpeed()
    if normalSpeed != 25 {
        t.Errorf("Expected speed 25, got %d", normalSpeed)
    }

    // Become encumbered
    for i := 0; i < 6; i++ {
        ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
    }

    encumberedSpeed := ent.GetSpeed()
    if encumberedSpeed != 15 {
        t.Errorf("Expected encumbered speed 15 (25-10), got %d", encumberedSpeed)
    }
}

func TestWornArmorBulkReduction(t *testing.T) {
    ent := entity.NewEntity("test", "Test", 1)

    // Chain mail is typically 2 Bulk, worn = 1 Bulk
    chainMail := item.NewArmor("chain_mail", "Chain Mail", item.MediumArmor, 5, 1, -2, -5, 14)
    chainMail.Bulk = 2
    ent.WornArmor = &chainMail

    // Worn armour bulk = 2 - 1 = 1 Bulk = 10 internal
    bulk := ent.CurrentBulk()
    if bulk.ToBulk() != 1 {
        t.Errorf("Worn chain mail should be 1 Bulk (reduced from 2), got %d", bulk.ToBulk())
    }
}
```

**Target File:** `pkg/rules/item/general_test.go`

```go
package item_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/item"
)

func TestGeneralItemCreation(t *testing.T) {
    rope := item.NewGeneralItem("rope", "Rope (50 ft)", item.CategoryAdventuringGear, 1, 100)

    if rope.Bulk != 1 {
        t.Errorf("Expected bulk 1, got %d", rope.Bulk)
    }
    if rope.Price != 100 {
        t.Errorf("Expected price 100cp, got %d", rope.Price)
    }
}

func TestBackpackCapacity(t *testing.T) {
    backpack := item.StandardItems["backpack"]

    if backpack.BulkCapacity != 40 { // 4 Bulk in internal units
        t.Errorf("Backpack should hold 4 Bulk worth, got %d", backpack.BulkCapacity/10)
    }
    if backpack.BulkReduction != 20 { // 2 Bulk reduction
        t.Errorf("Backpack should reduce 2 Bulk, got %d", backpack.BulkReduction/10)
    }
}
```

---

## 8. Execution Checklist

- [ ] Create `pkg/rules/entity/inventory.go` with Inventory and BulkValue types
- [ ] Add `Bulk` field to `Weapon` struct in `pkg/rules/item/weapon.go`
- [ ] Update `NewWeapon` function signature
- [ ] Update standard weapons registry with bulk values
- [ ] Create `pkg/rules/item/general.go` for GeneralItem type
- [ ] Add `Inventory` field to `Entity` struct
- [ ] Update `NewEntity()` to initialise Inventory
- [ ] Update `Entity.Clone()` to copy Inventory
- [ ] Add `BulkLimit()`, `MaxBulk()`, `CurrentBulk()` methods to Entity
- [ ] Add `UpdateEncumbranceConditions()` method
- [ ] Add `GetSpeedPenalty()` to ConditionTracker
- [ ] Update `GetSpeed()` to include encumbrance penalty
- [ ] Create `pkg/rules/entity/inventory_test.go`
- [ ] Create `pkg/rules/item/general_test.go`
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 9. CLI Commands

New/updated commands:

```bash
# View inventory
vd entity get paladin --field inventory
# Output:
# **Inventory:**
# | Item | Qty | Bulk |
# |------|-----|------|
# | Longsword | 1 | 1 |
# | Rope (50 ft) | 1 | 1 |
# | Torch | 5 | L |
#
# **Coins:** 50 gp, 23 sp, 15 cp
# **Total Bulk:** 3, 5L / 7 (Limit) / 12 (Max)
# **Status:** Normal

# Add item to inventory
vd inventory add paladin rope_50ft --quantity 2

# Remove item
vd inventory remove paladin torch --quantity 3

# Add coins
vd inventory coins paladin --add 100gp

# Check encumbrance
vd entity get paladin --field encumbrance
# Output:
# **Current Bulk:** 8
# **Bulk Limit:** 7 (5 + STR 2)
# **Max Bulk:** 12
# **Status:** Encumbered (-10 ft speed, Clumsy 1)
```

---

## 10. Container Logic (Optional Extension)

Containers like backpacks can hold items and reduce effective bulk. This is a stretch goal:

```go
type ContainerContents struct {
    ContainerID string
    Items       []InventoryItem
}

func (inv *Inventory) GetContainerBulkReduction() BulkValue {
    reduction := BulkValue(0)
    for _, item := range inv.Items {
        if gen, ok := item.Item.(*item.GeneralItem); ok {
            if gen.BulkReduction > 0 {
                // Items in container get bulk reduced
                reduction += BulkValue(gen.BulkReduction)
            }
        }
    }
    return reduction
}
```

This would require tracking which items are "inside" which containers, adding complexity. For MVP, assume backpacks grant a flat bulk reduction regardless of contents.
