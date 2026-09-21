package cleaningrecord

import "github.com/drainage/desilting/internal/shared/option"

// 天气选项取值。
const (
	WeatherSunny     = "sunny"
	WeatherCloudy    = "cloudy"
	WeatherOvercast  = "overcast"
	WeatherLightRain = "light_rain"
	WeatherHeavyRain = "heavy_rain"
)

// WeatherOptions 天气选项。
func WeatherOptions() []option.Option {
	return option.List(
		WeatherSunny, "晴",
		WeatherCloudy, "多云",
		WeatherOvercast, "阴",
		WeatherLightRain, "小雨",
		WeatherHeavyRain, "中到大雨",
	)
}
