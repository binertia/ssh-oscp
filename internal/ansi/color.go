package ansi

import "time"

// Color is a hex color string like "#00FFFF".
type Color string

// Matrix palette — Rain Room (cyberpunk / The Matrix)
const (
	MatrixBright Color = "#00FF41" // phosphor green
	MatrixGreen  Color = "#008F11" // terminal green
	MatrixDark   Color = "#004400" // faint trace
	MatrixDeep   Color = "#003300" // background residue
)

// Interstellar palette — Star Room (deep space telemetry)
const (
	StarWhite Color = "#E8E8E8" // cold white
	StarAmber Color = "#FFBF00" // NASA amber
	StarDim   Color = "#555555" // inactive
	StarDark  Color = "#1a1a1a" // void
)

// Blackhole palette — Gargantua / singularity
const (
	BHAccretion Color = "#FF4500" // orange-red
	BHBright    Color = "#FFFFFF" // white-hot
	BHDim       Color = "#8B0000" // dark red
	BHGlow      Color = "#FF8C00" // amber glow
)

// Arcane pentest palette — Red Team / cyberpunk fantasy
const (
	ArcaneRed  Color = "#FF0040" // neon crimson
	ArcaneGold Color = "#FFD700" // magic gold
	ArcanePink Color = "#FF00FF" // exploit neon
	ArcaneDim  Color = "#4B0082" // void purple
	ArcaneDeep Color = "#1a001a" // deep void
)

// Titan palette — Forgotten Robot / mechanical haunting
const (
	TitanSteel   Color = "#444455"
	TitanDim     Color = "#2a2a3a"
	TitanBright  Color = "#666677"
	TitanEyeOpen Color = "#00FFFF"
	TitanEyeDim  Color = "#004444"
	TitanCore    Color = "#FF4500"
	TitanCoreDim Color = "#8B0000"
	TitanRain    Color = "#333344"
	TitanRainLit Color = "#555566"
	TitanScan    Color = "#1a1a2a"
	TitanLight   Color = "#FFD700"
	TitanGlow    Color = "#FF8C00"
)

// Shared accent
const ColorWhite Color = "#FFFFFF"

// isNight returns true between 22:00 and 06:00.
func IsNight() bool {
	h := time.Now().Hour()
	return h >= 22 || h < 6
}

// SeasonalAccent returns a subtle tint color based on the current month.
func SeasonalAccent() Color {
	switch time.Now().Month() {
	case time.December, time.January, time.February:
		return "#88CCFF" // winter frost
	case time.March, time.April, time.May:
		return "#88FF88" // spring bud
	case time.June, time.July, time.August:
		return "#FFDD88" // summer heat
	case time.September, time.October, time.November:
		return "#FFAA44" // autumn amber
	}
	return ColorWhite
}

// SeasonalGlyph returns an ASCII symbol for the current season.
func SeasonalGlyph() rune {
	switch time.Now().Month() {
	case time.December, time.January, time.February:
		return '*' // snow
	case time.March, time.April, time.May:
		return '+' // bloom
	case time.June, time.July, time.August:
		return 'o' // sun
	case time.September, time.October, time.November:
		return '~' // leaf
	}
	return '·'
}

// Night palette shifts — ambient colors dim after dark.
const (
	MatrixGreenNight Color = "#006600"
	MatrixDarkNight  Color = "#002200"
	MatrixDeepNight  Color = "#001100"
	StarAmberNight   Color = "#CC9900"
	StarDimNight     Color = "#333333"
	BHDimNight       Color = "#550000"
	BHGlowNight      Color = "#CC6600"
	ArcaneGoldNight  Color = "#CCAA00"
	ArcaneDimNight   Color = "#2D0050"
)

// Dim returns nightColor if IsNight(), otherwise dayColor.
func Dim(dayColor, nightColor Color) Color {
	if IsNight() {
		return nightColor
	}
	return dayColor
}
