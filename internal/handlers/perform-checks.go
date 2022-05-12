package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/luksbutz/vigilate/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// HTTP is the unencrypted web service check
	HTTP = 1
	// HTTPS is the encrypted web service check
	HTTPS = 2
	// SSLCertificate is ssl certificate check
	SSLCertificate = 3
)

// jsonResp describes the JSON response sent back to client
type jsonResp struct {
	OK            bool      `json:"ok"`
	Message       string    `json:"message"`
	ServiceID     int       `json:"service_id"`
	HostServiceID int       `json:"host_service_id"`
	HostID        int       `json:"host_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	LastCheck     time.Time `json:"last_check"`
}

// ScheduledCheck performs a scheduled check on a host service by id
func (repo *DBRepo) ScheduledCheck(hostServiceID int) {
	log.Println("*********** Running check for", hostServiceID)

	hs, err := repo.DB.GetHostServiceByID(hostServiceID)
	if err != nil {
		log.Println(err)
		return
	}

	h, err := repo.DB.GetHostByID(hs.HostID)
	if err != nil {
		log.Println(err)
		return
	}

	// test the service
	msg, newStatus := repo.testServiceForHost(h, hs)

	if newStatus != hs.Status {
		repo.updateHostServiceStatusCount(h, hs, newStatus, msg)
	}
}

func (repo *DBRepo) updateHostServiceStatusCount(h models.Host, hs models.HostService, newStatus, msg string) {
	// update the host service record in the database with status (if changed) and last check
	hs.Status = newStatus
	hs.LastCheck = time.Now()

	err := repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		return
	}

	healthy, warning, problem, pending, err := repo.DB.GetAllServiceStatusCounts()
	if err != nil {
		log.Println(err)
		return
	}

	data := make(map[string]string)
	data["healthy_count"] = strconv.Itoa(healthy)
	data["warning_count"] = strconv.Itoa(warning)
	data["problem_count"] = strconv.Itoa(problem)
	data["pending_count"] = strconv.Itoa(pending)

	repo.broadcastMessage("public-channel", "host-service-count-changed", data)

	log.Println("Message is:", msg, "newStatus is:", newStatus)
}

// TestCheck manually tests a host service and sends JSON response
func (repo *DBRepo) TestCheck(w http.ResponseWriter, r *http.Request) {
	hostServiceID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	oldStatus := chi.URLParam(r, "oldStatus")
	okay := true

	// get host service
	hs, err := repo.DB.GetHostServiceByID(hostServiceID)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// get host?
	h, err := repo.DB.GetHostByID(hs.HostID)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// test the service
	msg, newStatus := repo.testServiceForHost(h, hs)

	// update the host service in the database (if changed) ant last check
	hs.Status = newStatus
	hs.UpdatedAt = time.Now()
	hs.LastCheck = time.Now()

	err = repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// broadcast service status changed event

	var resp jsonResp

	// create JSON
	if okay {
		resp = jsonResp{
			OK:            true,
			Message:       msg,
			ServiceID:     hs.ServiceID,
			HostServiceID: hs.ID,
			HostID:        hs.HostID,
			OldStatus:     oldStatus,
			NewStatus:     newStatus,
			LastCheck:     hs.LastCheck,
		}
	} else {
		resp.OK = false
		resp.Message = "Something went wrong"
	}

	// send JSON to client
	out, _ := json.MarshalIndent(resp, "", "\t")

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// testServiceForHost checks the service according to service id
func (repo *DBRepo) testServiceForHost(h models.Host, hs models.HostService) (string, string) {
	var msg, newStatus string

	switch hs.ServiceID {
	case HTTP:
		msg, newStatus = repo.testHTTPForHost(h.URL)
		break
	}

	if hs.Status != newStatus {
		data := map[string]string{
			"host_id":         strconv.Itoa(hs.HostID),
			"host_service_id": strconv.Itoa(hs.ID),
			"host_name":       h.HostName,
			"service_name":    hs.Service.ServiceName,
			"icon":            hs.Service.Icon,
			"status":          newStatus,
			"message":         fmt.Sprintf("%s on %s reports %s", hs.Service.ServiceName, h.HostName, newStatus),
			"last_check":      time.Now().Format("2006-01-02 15:04:05"),
		}

		repo.broadcastMessage("public-channel", "host-service-status-changed", data)
	}

	// TODO - send email/sms if appropriate

	return msg, newStatus
}

// testHTTPForHost tests HTTP service
func (repo *DBRepo) testHTTPForHost(url string) (string, string) {
	url = strings.TrimSuffix(url, "/")

	url = strings.Replace(url, "https://", "http://", -1)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("%s - %s", url, "error connecting"), "problem"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("%s - %s", url, resp.Status), "problem"
	}

	return fmt.Sprintf("%s - %s", url, resp.Status), "healthy"
}

func (repo *DBRepo) broadcastMessage(channel, messageType string, data map[string]string) {
	err := app.WsClient.Trigger(channel, messageType, data)
	if err != nil {
		log.Println(err)
	}
}
