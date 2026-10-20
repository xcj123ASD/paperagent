package config

import (
	"paperagent/internal/design"
	"paperagent/internal/model"
)

type Weights struct {
	Discipline float64 `json:"discipline"`
	Column     float64 `json:"column"`
	Quality    float64 `json:"quality"`
	Hotness    float64 `json:"hotness"`
}

func DefaultWeights() Weights {
	return Weights{Discipline: 0.35, Column: 0.30, Quality: 0.20, Hotness: 0.15}
}

func (w Weights) Valid() bool {
	s := w.Discipline + w.Column + w.Quality + w.Hotness
	return s >= 0.99 && s <= 1.01
}

type Config struct {
	Mock         bool
	Strict       bool
	Catalog      string
	Topic        string
	Weights      Weights
	BaseURL      string
	Key          string
	ModelScore   string
	ModelWrite   string
	Mode         model.Mode
	Design       *design.DesignInput
	PolishDesign bool
}