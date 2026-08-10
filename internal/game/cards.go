package game

import (
	"babibingo/internal/models"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
)

var (
	predefinedCards []models.CardJSON
	cardCacheOnce   sync.Once
)

// InitCardCache initializes the card cache
func InitCardCache() {
	cardCacheOnce.Do(func() {
		if err := LoadCardsFromJSON("data/cards.json"); err != nil {
			log.Printf("⚠️ Failed to load cards from JSON: %v, generating random cards", err)
			predefinedCards = make([]models.CardJSON, 0, 400)
			for i := 1; i <= 75; i++ {
				predefinedCards = append(predefinedCards, generateRandomCard(i))
			}
		}
		log.Printf("✅ Cards loaded: %d cards available", len(predefinedCards))
	})
}

// LoadCardsFromJSON loads cards from JSON file
func LoadCardsFromJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &predefinedCards)
}

// GetCardByID returns a card by its ID (1-75)
func GetCardByID(id int) (models.CardJSON, bool) {
	InitCardCache()
	
	for _, card := range predefinedCards {
		if card.CardID == id {
			return card, true
		}
	}
	
	log.Printf("⚠️ Card %d not found in cache, generating random card", id)
	return generateRandomCard(id), true
}

// GetAllCards returns all predefined cards
func GetAllCards() []models.CardJSON {
	InitCardCache()
	return predefinedCards
}

// checkWinPattern checks if a card has a winning pattern
func checkWinPattern(card models.CardJSON, marked []int) string {
	markedSet := make(map[int]bool)
	for _, n := range marked {
		markedSet[n] = true
	}

	// Build 5x5 grid
	grid := make([][]*int, 5)
	for i := range grid {
		grid[i] = make([]*int, 5)
	}

	// Fill grid correctly
	for i, n := range card.B {
		grid[i][0] = &n
	}
	for i, n := range card.I {
		grid[i][1] = &n
	}
	for i, n := range card.N {
		grid[i][2] = n // n is *int or nil
	}
	for i, n := range card.G {
		grid[i][3] = &n
	}
	for i, n := range card.O {
		grid[i][4] = &n
	}

	// Helper to check if a cell is marked
	isMarked := func(row, col int) bool {
		val := grid[row][col]
		if val == nil {
			return true // Free space is always marked
		}
		return markedSet[*val]
	}

	// ✅ Check horizontal wins
	for row := 0; row < 5; row++ {
		win := true
		for col := 0; col < 5; col++ {
			if !isMarked(row, col) {
				win = false
				break
			}
		}
		if win {
			return "horizontal"
		}
	}

	// ✅ Check vertical wins
	for col := 0; col < 5; col++ {
		win := true
		for row := 0; row < 5; row++ {
			if !isMarked(row, col) {
				win = false
				break
			}
		}
		if win {
			return "vertical"
		}
	}

	// ✅ Check diagonal wins (top-left to bottom-right)
	win := true
	for i := 0; i < 5; i++ {
		if !isMarked(i, i) {
			win = false
			break
		}
	}
	if win {
		return "diagonal"
	}

	// ✅ Check diagonal wins (top-right to bottom-left)
	win = true
	for i := 0; i < 5; i++ {
		if !isMarked(i, 4-i) {
			win = false
			break
		}
	}
	if win {
		return "diagonal"
	}

	// ✅ NEW: Check Four Corners
	// Corners: (0,0), (0,4), (4,0), (4,4)
	corners := [][2]int{{0, 0}, {0, 4}, {4, 0}, {4, 4}}
	allCornersMarked := true
	for _, pos := range corners {
		if !isMarked(pos[0], pos[1]) {
			allCornersMarked = false
			break
		}
	}
	if allCornersMarked {
		return "four_corners"
	}

	// ✅ NEW: Check Center Cross
	// Center cross: middle row (2,0-4) and middle column (0-4,2)
	centerCross := [][2]int{
		 {2, 1}, {2, 2}, {2, 3}, // Middle row
		 {1, 2}, {3, 2}, // Middle column (excluding center which is already counted)
	}
	allCrossMarked := true
	for _, pos := range centerCross {
		if !isMarked(pos[0], pos[1]) {
			allCrossMarked = false
			break
		}
	}
	if allCrossMarked {
		return "center_cross"
	}

	return ""
}

// verifyWinDoubleCheck verifies a win by counting marked cells in the pattern
func verifyWinDoubleCheck(card models.CardJSON, marked []int, pattern string) bool {
	markedSet := make(map[int]bool)
	for _, n := range marked {
		markedSet[n] = true
	}

	// Build grid
	grid := make([][]*int, 5)
	for i := range grid {
		grid[i] = make([]*int, 5)
	}
	for i, n := range card.B {
		grid[i][0] = &n
	}
	for i, n := range card.I {
		grid[i][1] = &n
	}
	for i, n := range card.N {
		grid[i][2] = n
	}
	for i, n := range card.G {
		grid[i][3] = &n
	}
	for i, n := range card.O {
		grid[i][4] = &n
	}

	isMarked := func(row, col int) bool {
		val := grid[row][col]
		if val == nil {
			return true
		}
		return markedSet[*val]
	}

	switch pattern {
	case "horizontal":
		for row := 0; row < 5; row++ {
			markedCount := 0
			for col := 0; col < 5; col++ {
				if isMarked(row, col) {
					markedCount++
				}
			}
			if markedCount == 5 {
				return true
			}
		}
		return false

	case "vertical":
		for col := 0; col < 5; col++ {
			markedCount := 0
			for row := 0; row < 5; row++ {
				if isMarked(row, col) {
					markedCount++
				}
			}
			if markedCount == 5 {
				return true
			}
		}
		return false

	case "diagonal":
		// Top-left to bottom-right
		markedCount := 0
		for i := 0; i < 5; i++ {
			if isMarked(i, i) {
				markedCount++
			}
		}
		if markedCount == 5 {
			return true
		}
		// Top-right to bottom-left
		markedCount = 0
		for i := 0; i < 5; i++ {
			if isMarked(i, 4-i) {
				markedCount++
			}
		}
		return markedCount == 5

	case "four_corners":
		corners := [][2]int{{0, 0}, {0, 4}, {4, 0}, {4, 4}}
		for _, pos := range corners {
			if !isMarked(pos[0], pos[1]) {
				return false
			}
		}
		return true

	case "center_cross":
		centerCross := [][2]int{
			{2, 1}, {2, 2}, {2, 3},
			{1, 2}, {3, 2},
		}
		for _, pos := range centerCross {
			if !isMarked(pos[0], pos[1]) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

// generateRandomCard generates a random bingo card
func generateRandomCard(cardID int) models.CardJSON {
	return models.CardJSON{
		B:      pickRandom(1, 15, 5),
		I:      pickRandom(16, 30, 5),
		N:      appendNFreeSpace(pickRandom(31, 45, 4)),
		G:      pickRandom(46, 60, 5),
		O:      pickRandom(61, 75, 5),
		CardID: cardID,
	}
}

// pickRandom picks random numbers
func pickRandom(min, max, count int) []int {
	nums := make([]int, max-min+1)
	for i := range nums {
		nums[i] = min + i
	}
	rand.Shuffle(len(nums), func(i, j int) { nums[i], nums[j] = nums[j], nums[i] })
	result := nums[:count]
	sort.Ints(result)
	return result
}

// getBingoLetter returns the letter for a bingo number
func getBingoLetter(num int) string {
	switch {
	case num >= 1 && num <= 15:
		return "B"
	case num >= 16 && num <= 30:
		return "I"
	case num >= 31 && num <= 45:
		return "N"
	case num >= 46 && num <= 60:
		return "G"
	case num >= 61 && num <= 75:
		return "O"
	default:
		return ""
	}
}

// containsNumber checks if a number is in a card
func containsNumber(card models.CardJSON, num int) bool {
	for _, n := range card.B {
		if n == num {
			return true
		}
	}
	for _, n := range card.I {
		if n == num {
			return true
		}
	}
	for _, n := range card.N {
		if n != nil && *n == num {
			return true
		}
	}
	for _, n := range card.G {
		if n == num {
			return true
		}
	}
	for _, n := range card.O {
		if n == num {
			return true
		}
	}
	return false
}

// appendNFreeSpace appends free space to N column
func appendNFreeSpace(nums []int) []*int {
	result := make([]*int, 5)
	result[0] = &nums[0]
	result[1] = &nums[1]
	result[2] = nil // Free space
	result[3] = &nums[2]
	result[4] = &nums[3]
	return result
}

// maskPhone masks a phone number
func maskPhone(phone string) string {
	if len(phone) < 8 {
		return phone
	}
	return phone[:4] + "****" + phone[len(phone)-2:]
}