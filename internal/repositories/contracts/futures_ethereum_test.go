package contracts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetOngoingDeliveryRangeBeforeFirstDeliveryDate(t *testing.T) {
	firstDeliveryDate := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	deliveryInterval := 7 * 24 * time.Hour
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	start, end := GetOngoingDeliveryRange(firstDeliveryDate, deliveryInterval, now)

	require.Equal(t, firstDeliveryDate, start)
	require.Equal(t, firstDeliveryDate.Add(deliveryInterval), end)
}

func TestGetOngoingDeliveryRangeWithin1stDeliveryRange(t *testing.T) {
	firstDeliveryDate := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	deliveryInterval := 7 * 24 * time.Hour
	now := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	start, end := GetOngoingDeliveryRange(firstDeliveryDate, deliveryInterval, now)

	require.Equal(t, firstDeliveryDate, start)
	require.Equal(t, firstDeliveryDate.Add(deliveryInterval), end)
}

func TestGetOngoingDeliveryRangeWithin2ndDeliveryRange(t *testing.T) {
	firstDeliveryDate := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	deliveryInterval := 7 * 24 * time.Hour
	now := time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC)
	start, end := GetOngoingDeliveryRange(firstDeliveryDate, deliveryInterval, now)

	require.Equal(t, firstDeliveryDate.Add(deliveryInterval), start)
	require.Equal(t, firstDeliveryDate.Add(deliveryInterval*2), end)
}
