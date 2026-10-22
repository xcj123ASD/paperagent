package retriever

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"paperagent/internal/model"
)

type DisciplineWeights struct{ Discipline, Column, Quality, Hotness float64 }

func MatchJournals(topic, discipline, sub string, keywords []string, catalog []model.Journal, w DisciplineWeights) []model.Journal {
	if len(catalog) == 0 {
		return nil
	}
	kw := lowerSet(keywords)
	items := make([]struct {
		j          model.Journal
		d, c, q, h float64
	}, len(catalog))
	for i, j := range catalog {
		items[i].j = j
		items[i].d = discScore(j, discipline, sub, kw)
		items[i].c = columnScore(j, kw)
		items[i].q = qualityScore(j.Level)
		items[i].h = hotnessScore(kw)
	}
	mn := func(f func(int) float64) (float64, float64) {
		a, b := math.MaxFloat64, -math.MaxFloat64
		for i := range items {
			v := f(i)
			if v < a {
				a = v
			}
			if v > b {
				b = v
			}
		}
		return a, b
	}
	d0, d1 := mn(func(i int) float64 { return items[i].d })
	c0, c1 := mn(func(i int) float64 { return items[i].c })
	q0, q1 := mn(func(i int) float64 { return items[i].q })
	h0, h1 := mn(func(i int) float64 { return items[i].h })
	norm := func(v, a, b float64) float64 {
		if b <= a {
			return 0.5
		}
		return (v - a) / (b - a)
	}
	for i := range items {
		x := &items[i]
		x.j.Score = round(w.Discipline*norm(x.d, d0, d1) + w.Column*norm(x.c, c0, c1) +
			w.Quality*norm(x.q, q0, q1) + w.Hotness*norm(x.h, h0, h1))
		x.j.Breakdown = fmt.Sprintf("学科=%.2f 栏目=%.2f 质量=%.2f 时效=%.2f", x.d, x.c, x.q, x.h)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].j.Score > items[j].j.Score })
	out := make([]model.Journal, len(items))
	for i := range items {
		out[i] = items[i].j
	}
	return out
}

func discScore(j model.Journal, disc, sub string, kw map[string]bool) float64 {
	if strings.Contains(j.Sub, sub) || strings.Contains(sub, j.Sub) {
		return 1.0
	}
	if strings.Contains(j.Discipline, disc) {
		return 0.8
	}
	for k := range kw {
		if strings.Contains(j.Sub, k) || strings.Contains(j.Discipline, k) {
			return 0.5
		}
	}
	return 0.1
}

func columnScore(j model.Journal, kw map[string]bool) float64 {
	if len(kw) == 0 {
		return 0.5
	}
	hit := 0
	for _, c := range j.Columns {
		for k := range kw {
			if strings.Contains(c, k) {
				hit++
				break
			}
		}
	}
	return float64(hit) / float64(len(kw))
}

func qualityScore(lv string) float64 {
	switch lv {
	case "CSSCI":
		return 1.0
	case "CSSCI扩展":
		return 0.85
	case "北大核心":
		return 0.7
	default:
		return 0.4
	}
}

func hotnessScore(kw map[string]bool) float64 {
	hot := []string{"数字", "智能", "大模型", "治理", "低碳", "生成式"}
	n := 0
	for k := range kw {
		for _, h := range hot {
			if strings.Contains(k, h) {
				n++
				break
			}
		}
	}
	if len(kw) == 0 {
		return 0.5
	}
	return float64(n) / float64(len(kw))
}

func lowerSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[strings.TrimSpace(s)] = true
	}
	return m
}

func round(f float64) float64 { return math.Round(f*100) / 100 }
