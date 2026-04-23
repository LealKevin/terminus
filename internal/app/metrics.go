package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalMessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "terminus_messages_received_total",
		Help: "The total number of messages received from clients",
	}, []string{"type"})

	totalMessagesErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "terminus_messages_errors_total",
		Help: "The total number of errors processing messages from clients",
	}, []string{"type"})

	messageProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "terminus_message_processing_duration_seconds",
		Help: "Duration of message processing",
	}, []string{"type"})

	playersOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_players_online",
		Help: "Total of players online",
	})

	playerMoveDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_player_move_duration_seconds",
		Help: "Latency of HandlePlayerMove",
	})

	playerMoveInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_player_move_in_flight",
		Help: "In-flight player move operations",
	})

	playerMoves = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_player_moves_total",
		Help: "Total player moves",
	})

	playerAttacks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_player_attacks_total",
		Help: "Total player attacks",
	})

	playerAttackDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_player_attack_duration_seconds",
		Help: "Latency of HandlePlayerAttack",
	})

	playerAttackInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_player_attack_in_flight",
		Help: "In-flight player attack operations",
	})

	mobMoves = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_mob_moves_total",
		Help: "Total mob moves",
	})

	mobMoveDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_mob_move_duration_seconds",
		Help: "Latency of HandleMobMove",
	})

	mobMoveInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_mob_move_in_flight",
		Help: "In-flight mob move operations",
	})
)
