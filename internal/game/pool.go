package game

// GetPoolBreakdown returns net pool and house cut
func GetPoolBreakdown(grossPool float64) (netPool, houseCut float64) {
	netPool = grossPool * (1 - HouseCutPercent)
	houseCut = grossPool * HouseCutPercent
	return netPool, houseCut
}

// CalculateNetPool returns the prize pool after house cut
func CalculateNetPool(grossPool float64) float64 {
	return grossPool * (1 - HouseCutPercent)
}

// CalculateHouseCut returns the house cut amount
func CalculateHouseCut(grossPool float64) float64 {
	return grossPool * HouseCutPercent
}

// UpdatePool updates the gross pool based on reserved cards
func (e *Engine) UpdatePool(state *GameState) {
	state.Game.TotalPool = float64(len(state.ReservedCards)) * StakeAmount
}