package handler

import (
	"net/http"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/config"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/database"
)

// APIDashboard handles GET /api/v1/dashboard.
// Returns stats, services health, and recent emails.
func (h *Handler) APIDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, _ := h.DB.GetEmailStats()
	if stats == nil {
		stats = &database.EmailStats{}
	}

	recent, _ := h.DB.RecentEmails(20)
	if recent == nil {
		recent = []database.EmailLog{}
	}

	services := config.CheckServices()

	h.jsonOK(w, map[string]any{
		"stats": map[string]int{
			"sent_24h":   stats.Sent24h,
			"sent_7d":    stats.Sent7d,
			"failed_24h": stats.Failed24h,
			"queued":     stats.Queued,
		},
		"services": services,
		"recent":   recent,
	})
}
