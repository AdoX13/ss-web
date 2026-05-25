package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/ocr"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/routes"
	"mqtt-streaming-server/utils"
)

type BrokerHandler struct {
	photoRepository      domain.PhotoRepository
	deviceRepository     domain.DeviceRepository
	reviewItemRepository domain.ReviewItemRepository
	medicalRepository    *repository.MedicalRepository
	ocrClient            ocr.Client
	reviewNotifyCh       chan<- *domain.ReviewItem // may be nil
}

// NewBrokerHandler creates a broker handler. reviewNotifyCh receives new
// ReviewItems for WebSocket broadcast; pass nil to disable notifications.
func NewBrokerHandler(db *mongo.Database, ocrClient ocr.Client, reviewNotifyCh chan<- *domain.ReviewItem) BrokerHandler {
	medicalRepo, err := repository.NewMedicalRepositoryFromEnv(db)
	if err != nil {
		slog.Warn("encrypted medical projection disabled; set MEDSEC_MASTER_KEY to enable P6 PHI storage", "err", err)
	}
	return BrokerHandler{
		photoRepository:      repository.NewPhotoRepository(db),
		deviceRepository:     repository.NewDeviceRepository(db),
		reviewItemRepository: repository.NewReviewItemRepository(db),
		medicalRepository:    medicalRepo,
		ocrClient:            ocrClient,
		reviewNotifyCh:       reviewNotifyCh,
	}
}

func (b BrokerHandler) HandlePhoto(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	var deviceID string
	if topic == "ssproject/images" {
		deviceID = "camera_stream"
	} else if len(topic) > len("ssproject/images/") {
		deviceID = topic[len("ssproject/images/"):]
	} else {
		deviceID = "unknown"
	}

	ctx := context.Background()
	slog.Info("received photo", "topic", msg.Topic())

	device, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			slog.Info("auto-registering unknown device", "device_id", deviceID)
			newDevice := &domain.Device{
				DeviceID:     deviceID,
				DeviceName:   "Unknown Device (" + deviceID + ")",
				DeviceStatus: "active",
			}
			if err := b.deviceRepository.Save(ctx, newDevice); err != nil {
				slog.Error("failed to auto-register device", "device_id", deviceID, "err", err)
				return
			}
			device = newDevice
		} else {
			slog.Error("failed to look up device", "device_id", deviceID, "err", err)
			return
		}
	}
	slog.Info("processing photo", "device", device.DeviceName)

	body := msg.Payload()
	_, imageType, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to decode image header", "err", err)
		return
	}

	// Generate the photo id up front so the OCR result and any review items
	// share the same identifier.
	photoID := primitive.NewObjectID()

	// Extract text from the image via the sandboxed OCR worker.
	startTime := time.Now()
	ocrCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	result, err := b.ocrClient.Process(ocrCtx, photoID.Hex(), body)
	cancel()
	processingDuration := time.Since(startTime).Milliseconds()
	utils.OcrJobsTotal.Inc()
	utils.OcrProcessingMsSum.Add(float64(processingDuration))
	utils.OcrProcessingMsCount.Inc()

	if err != nil {
		slog.Error("ocr worker error", "photo_id", photoID.Hex(), "err", err)
		// Fall back to saving the image with empty OCR data rather than
		// dropping the message entirely.
		result = nil
	}

	photo := b.buildPhoto(photoID, deviceID, imageType, result)

	if err := b.photoRepository.Save(ctx, photo); err != nil {
		slog.Error("failed to save photo", "photo_id", photoID.Hex(), "err", err)
		return
	}
	if err := b.medicalRepository.SaveFromPhoto(ctx, photo); err != nil {
		slog.Warn("failed to save encrypted medical projection", "photo_id", photoID.Hex(), "err", err)
	}

	if result != nil && result.NeedsReview {
		b.createReviewItems(ctx, photoID, result)
	}

	keyName := fmt.Sprintf("photos/%d.%s", photo.Timestamp.Unix(), imageType)
	if err := utils.SaveToLocal(body, keyName); err != nil {
		slog.Error("failed to save photo locally", "key", keyName, "err", err)
	}
}

// buildPhoto maps an OCR result (may be nil on worker error) onto a Photo.
func (b BrokerHandler) buildPhoto(id primitive.ObjectID, deviceID, imageType string, result *ocr.Result) *domain.Photo {
	photo := &domain.Photo{
		ID:        id,
		ImageType: imageType,
		Timestamp: time.Now().UTC(),
		DeviceID:  deviceID,
	}
	if result == nil {
		photo.Text = "OCR unavailable"
		return photo
	}

	photo.Text = result.RawText
	photo.OverallConfidence = result.OverallConfidence
	photo.NeedsReview = result.NeedsReview

	f := result.Fields
	if f.PatientName != nil && f.PatientName.Value != nil {
		photo.Nume = *f.PatientName.Value
	}
	if f.PatientCNP != nil && f.PatientCNP.Value != nil {
		photo.CNP = *f.PatientCNP.Value
	}
	if f.Profession != nil && f.Profession.Value != nil {
		photo.ProfesieFunctie = *f.Profession.Value
	}
	if f.Workplace != nil && f.Workplace.Value != nil {
		photo.LocDeMunca = *f.Workplace.Value
	}
	if f.ControlType != nil && f.ControlType.Value != nil {
		photo.TipControl = *f.ControlType.Value
		photo.ControlAngajare = *f.ControlType.Value == "Angajare"
		photo.ControlPeriodic = *f.ControlType.Value == "Periodic"
		photo.ControlAdaptare = *f.ControlType.Value == "Adaptare"
		photo.ControlReluare = *f.ControlType.Value == "Reluare"
		photo.ControlSupraveghere = *f.ControlType.Value == "Supraveghere"
		photo.ControlAlte = *f.ControlType.Value == "Alte"
	}
	if f.MedicalOpinion != nil && f.MedicalOpinion.Value != nil {
		photo.AvizMedical = *f.MedicalOpinion.Value
		photo.AvizApt = *f.MedicalOpinion.Value == "APT"
		photo.AvizAptConditionat = *f.MedicalOpinion.Value == "APT Condiționat"
		photo.AvizInaptTemporar = *f.MedicalOpinion.Value == "Inapt Temporar"
		photo.AvizInapt = *f.MedicalOpinion.Value == "Inapt"
	}
	if f.ExamDate != nil && f.ExamDate.Value != nil {
		if t, err := parseRomanianDate(*f.ExamDate.Value); err == nil {
			photo.Data = t
		}
	}
	if f.ExpirationDate != nil && f.ExpirationDate.Value != nil {
		if t, err := parseRomanianDate(*f.ExpirationDate.Value); err == nil {
			photo.DataUrmExaminari = t
		}
	}
	if f.DoctorName != nil && f.DoctorName.Value != nil {
		photo.DoctorName = *f.DoctorName.Value
	}
	return photo
}

// createReviewItems writes one review_items entry per field whose confidence
// is below the review threshold or whose value is nil.
func (b BrokerHandler) createReviewItems(ctx context.Context, photoID primitive.ObjectID, result *ocr.Result) {
	type entry struct {
		name  string
		value *string
		conf  float64
	}
	f := result.Fields
	strPtr := func(ef *ocr.EnumField) *string {
		if ef == nil {
			return nil
		}
		return ef.Value
	}
	entries := []entry{
		{"patient_name", fieldVal(f.PatientName), fieldConf(f.PatientName)},
		{"patient_cnp", fieldVal(f.PatientCNP), fieldConf(f.PatientCNP)},
		{"profession", fieldVal(f.Profession), fieldConf(f.Profession)},
		{"workplace", fieldVal(f.Workplace), fieldConf(f.Workplace)},
		{"control_type", strPtr(f.ControlType), enumConf(f.ControlType)},
		{"medical_opinion", strPtr(f.MedicalOpinion), enumConf(f.MedicalOpinion)},
		{"exam_date", fieldVal(f.ExamDate), fieldConf(f.ExamDate)},
		{"expiration_date", fieldVal(f.ExpirationDate), fieldConf(f.ExpirationDate)},
		{"doctor_name", fieldVal(f.DoctorName), fieldConf(f.DoctorName)},
	}
	now := time.Now().UTC()
	for _, e := range entries {
		if e.value != nil && e.conf >= ocr.OverallConfidenceReviewThreshold {
			continue
		}
		item := &domain.ReviewItem{
			ImageID:            photoID,
			FieldName:          e.name,
			OriginalValue:      e.value,
			OriginalConfidence: e.conf,
			Status:             domain.ReviewItemPending,
			CreatedAt:          now,
		}
		if err := b.reviewItemRepository.Save(ctx, item); err != nil {
			slog.Error("failed to save review item", "photo_id", photoID.Hex(), "field", e.name, "err", err)
		} else {
			if b.reviewNotifyCh != nil {
				select {
				case b.reviewNotifyCh <- item:
				default:
				}
			}
			routes.IncrReviewItems()
		}
	}
}

func fieldVal(f *ocr.Field) *string {
	if f == nil {
		return nil
	}
	return f.Value
}

func fieldConf(f *ocr.Field) float64 {
	if f == nil {
		return 0
	}
	return f.Confidence
}

func enumConf(f *ocr.EnumField) float64 {
	if f == nil {
		return 0
	}
	return f.Confidence
}

// parseRomanianDate accepts DD.MM.YYYY and DD/MM/YYYY.
func parseRomanianDate(s string) (time.Time, error) {
	for _, layout := range []string{"02.01.2006", "02/01/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date format: %q", s)
}

func (b BrokerHandler) RegisterDevice(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	deviceID := topic[len("register/"):]
	ctx := context.Background()
	slog.Info("device registration", "topic", msg.Topic())
	body := msg.Payload()

	var deviceName, ipAddress, port string
	var registration struct {
		Name string `json:"name"`
		IP   string `json:"ip"`
		Port string `json:"port"`
	}
	if err := json.Unmarshal(body, &registration); err == nil && registration.Name != "" {
		deviceName = registration.Name
		ipAddress = registration.IP
		port = registration.Port
	} else {
		deviceName = string(body)
	}

	_, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil && err != mongo.ErrNoDocuments {
		slog.Error("failed to check device", "device_id", deviceID, "err", err)
		return
	}
	if err == mongo.ErrNoDocuments {
		err = b.deviceRepository.Save(ctx, &domain.Device{
			DeviceID:     deviceID,
			DeviceName:   deviceName,
			DeviceStatus: "active",
			IPAddress:    ipAddress,
			Port:         port,
			LastSeen:     time.Now().UTC(),
		})
		if err != nil {
			slog.Error("failed to register device", "device_id", deviceID, "err", err)
			return
		}
		slog.Info("device registered", "device_id", deviceID, "ip", ipAddress)
		return
	}
	err = b.deviceRepository.Update(ctx, deviceID, &domain.Device{
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		DeviceStatus: "active",
		IPAddress:    ipAddress,
		Port:         port,
		LastSeen:     time.Now().UTC(),
	})
	if err != nil {
		slog.Error("failed to update device", "device_id", deviceID, "err", err)
	}
	slog.Info("device updated", "device_id", deviceID)
}

func (b BrokerHandler) DisconnectDevice(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	var deviceID string
	if len(topic) > len("device/id/") {
		deviceID = topic[len("device/id/"):]
	} else {
		return
	}

	ctx := context.Background()
	message := string(msg.Payload())
	if message != "Device Disconnected" {
		return
	}

	device, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil {
		return
	}
	if device.DeviceStatus != "active" {
		return
	}
	if err := b.deviceRepository.Update(ctx, deviceID, &domain.Device{
		DeviceID:     deviceID,
		DeviceStatus: "inactive",
		DeviceName:   device.DeviceName,
	}); err != nil {
		slog.Error("failed to mark device inactive", "device_id", deviceID, "err", err)
	}
}

func (b BrokerHandler) HandleCommand(_ mqtt.Client, msg mqtt.Message) {
	slog.Info("received command", "topic", msg.Topic(), "payload", string(msg.Payload()))
}
