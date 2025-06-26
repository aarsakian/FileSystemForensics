package attributes

type DevStatsItem struct {
}

func (devStats *DevStatsItem) Parse(data []byte) int {
	return len(data)
}

func (devStats DevStatsItem) GetInfo() string {
	return ""
}

func (devStats DevStatsItem) ShowInfo() {

}
