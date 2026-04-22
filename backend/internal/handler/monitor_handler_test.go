package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestBuildModelShareUsesActualCostForOrderingAndShare(t *testing.T) {
	items := buildModelShare([]usagestats.ModelStat{
		{
			Model:       "cheap-heavy",
			Requests:    10,
			TotalTokens: 10_000,
			ActualCost:  2,
		},
		{
			Model:       "expensive-light",
			Requests:    2,
			TotalTokens: 500,
			ActualCost:  8,
		},
	}, 10)

	require.Len(t, items, 2)
	require.Equal(t, "expensive-light", items[0].Model)
	require.InDelta(t, 80.0, items[0].SharePercent, 0.0001)
	require.Equal(t, "cheap-heavy", items[1].Model)
	require.InDelta(t, 20.0, items[1].SharePercent, 0.0001)
}
