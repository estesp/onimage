package services

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/estesp/onimage/pkg/util"
	"github.com/sirupsen/logrus"
)

const retries = 4

type WeatherData struct {
	appId      string
	locationId int
	baseURL    string
	units      string
	errChan    chan error
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

func NewWeatherDataService(config map[string]interface{}, errChan chan error) (*WeatherData, error) {
	baseUrl, err := util.GetStringFromConfig(config, "weather.base_url")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'weather.base_url' from config: %w", err)
	}
	locationId, err := util.GetIntFromConfig(config, "weather.location_code")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'weather.location_code' from config: %w", err)
	}
	appId, err := util.GetStringFromConfig(config, "weather.appid")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'weather.appid' from config: %w", err)
	}
	units, err := util.GetStringFromConfig(config, "weather.units")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'weather.units' from config: %w", err)
	}
	return &WeatherData{
		appId:      appId,
		locationId: int(locationId),
		baseURL:    baseUrl,
		units:      units,
		errChan:    errChan,
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
