package services

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/sirupsen/logrus"
)

const retries = 4

type WeatherData struct {
	appId      string
	locationId int
	baseURL    string
	units      string
}

type Params struct {
	Id    int    `url:"id,omitempty"`
	AppId string `url:"appid,omitempty"`
	Units string `url:"units,omitempty"`
}

type MainSection struct {
	FeelsLike float32 `json:"feels_like"`
	Humidity  float32 `json:"humidity"`
	Pressure  int     `json:"pressure"`
	Temp      float32 `json:"temp"`
	TempMax   float32 `json:"temp_max"`
	TempMin   float32 `json:"temp_min"`
}

type SysSection struct {
	Sunrise int64 `json:"sunrise"`
	Sunset  int64 `json:"sunset"`
}

type WeatherDesc struct {
	Description string `json:"description"`
	Icon        string `json:"icon"`
	ID          int    `json:"id"`
	Main        string `json:"main"`
}

type WindSection struct {
	Deg   int     `json:"deg"`
	Gust  float64 `json:"gust"`
	Speed float64 `json:"speed"`
}

type Weather struct {
	Main        MainSection   `json:"main"`
	Sys         SysSection    `json:"sys"`
	WeatherDesc []WeatherDesc `json:"weather"`
	Wind        WindSection   `json:"wind"`
}

func NewWeatherDataService(config map[string]interface{}) (*WeatherData, error) {
	baseUrl, ok := config["weather.base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'weather.base_url'")
	}
	locationId, ok := config["weather.location_code"].(int)
	if !ok {
		return nil, fmt.Errorf("config file has no integer entry for 'weather.location_code'")
	}
	appId, ok := config["weather.appid"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'weather.appid'")
	}
	units, ok := config["weather.units"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'weather.units'")
	}
	return &WeatherData{
		appId:      appId,
		locationId: locationId,
		baseURL:    baseUrl,
		units:      units,
	}, nil
}

func (w *WeatherData) GetCurrentWeather() (*Weather, error) {

	var err error

	params := &Params{
		Id:    w.locationId,
		AppId: w.appId,
		Units: w.units,
	}

	var toClient = &http.Client{
		Timeout: time.Second * 10,
	}

	weather := new(Weather)

	for i := 0; i < retries; i++ {
		_, err = sling.New().Client(toClient).Get(w.baseURL).QueryStruct(params).ReceiveSuccess(weather)
		if err != nil {
			logrus.Infof("Try %d: failed to query openweathermap for current conditions: %v", i+1, err)
			time.Sleep(time.Duration(int(math.Pow(float64(i+1), 2))) * time.Second)
			continue
		}
		return weather, nil
	}
	return nil, fmt.Errorf("weather conditions failed after %d retries calling openweathermap: %w", retries, err)
}

func (w *WeatherData) GetCurrentTempStr() (string, error) {
	wdata, err := w.GetCurrentWeather()
	if err != nil {
		return "", fmt.Errorf("couldn't retrieve weather: %w", err)
	}
	return fmt.Sprintf("%2.1f", wdata.Main.Temp), nil
}
