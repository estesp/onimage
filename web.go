package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type suntimez struct {
	Photo      int    `json:"photo"`
	Sunrise    int64  `json:"sunrise"`
	Sunset     int64  `json:"sunset"`
	NowUnix    int64  `json:"now_unix"`
	SunriseStr string `json:"sunrise_str"`
	SunsetStr  string `json:"sunset_str"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := r.Context().Value("ps").(*ProcessingService)
	now := time.Now()
	sunrisePre := p.GetSunrise() - 1800
	sunsetPost := p.GetSunset() + 1800
	resp := 0
	if now.Unix() >= sunrisePre && now.Unix() <= sunsetPost {
		resp = 1
	}
	riseTime := time.Unix(p.GetSunrise(), 0)
	setTime := time.Unix(p.GetSunset(), 0)
	sresp := suntimez{
		Photo:      resp,
		Sunrise:    p.GetSunrise(),
		Sunset:     p.GetSunset(),
		NowUnix:    now.Unix(),
		SunriseStr: riseTime.String(),
		SunsetStr:  setTime.String(),
	}
	b, err := json.Marshal(sresp)
	if err != nil {
		logrus.Errorf("can't marshal suntimes JSON: %v", err)
	}
	w.Write(b)
}

func (p *ProcessingService) StartWebHandler() {
	mux := http.NewServeMux()
	mux.HandleFunc("/phototimez", handler)
	contextHandler := addContext(mux, p)
	logrus.Fatal(http.ListenAndServe(":5000", contextHandler))
}

func addContext(next http.Handler, p *ProcessingService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Add data to context
		ctx := context.WithValue(r.Context(), "ps", p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
