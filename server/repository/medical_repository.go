package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	medcrypto "mqtt-streaming-server/crypto"
	"mqtt-streaming-server/domain"
)

type MedicalRepository struct {
	db     *mongo.Database
	master []byte
}

func NewMedicalRepositoryFromEnv(db *mongo.Database) (*MedicalRepository, error) {
	master, err := medcrypto.LoadMasterKeyFromEnv()
	if err != nil {
		return nil, err
	}
	return &MedicalRepository{db: db, master: master}, nil
}

// SaveFromPhoto writes the P6 normalized, encrypted projection of a legacy
// Photo. The legacy photos collection remains for compatibility with Lab 1 UI.
func (r *MedicalRepository) SaveFromPhoto(ctx context.Context, photo *domain.Photo) error {
	if r == nil {
		return nil
	}
	if photo == nil {
		return errors.New("photo is required")
	}

	now := time.Now().UTC()
	patientID, err := r.upsertPatient(ctx, photo, now)
	if err != nil {
		return err
	}

	record := bson.M{
		"image_id":         photo.ID,
		"control_type":    photo.TipControl,
		"medical_opinion": photo.AvizMedical,
		"profession":      photo.ProfesieFunctie,
		"created_at":      now,
	}
	if !photo.Data.IsZero() {
		record["exam_date"] = photo.Data
	}
	if !photo.DataUrmExaminari.IsZero() {
		record["expiration_date"] = photo.DataUrmExaminari
	}
	if patientID != primitive.NilObjectID {
		record["patient_id"] = patientID
	}
	if strings.TrimSpace(photo.LocDeMunca) != "" {
		env, err := medcrypto.EncryptString(r.master, "medical_records.workplace", photo.LocDeMunca)
		if err != nil {
			return err
		}
		record["workplace"] = env
	}
	if strings.TrimSpace(photo.DoctorName) != "" {
		env, err := medcrypto.EncryptString(r.master, "medical_records.doctor_name", photo.DoctorName)
		if err != nil {
			return err
		}
		record["doctor_name"] = env
	}

	_, err = r.db.Collection("medical_records").InsertOne(ctx, record)
	return err
}

func (r *MedicalRepository) upsertPatient(ctx context.Context, photo *domain.Photo, now time.Time) (primitive.ObjectID, error) {
	name := strings.TrimSpace(strings.TrimSpace(photo.Nume) + " " + strings.TrimSpace(photo.Prenume))
	cnp := medcrypto.NormalizeCNP(photo.CNP)
	if name == "" && cnp == "" {
		return primitive.NilObjectID, nil
	}

	doc := bson.M{"created_at": now}
	filter := bson.M{}
	if name != "" {
		env, err := medcrypto.EncryptString(r.master, "patients.name", name)
		if err != nil {
			return primitive.NilObjectID, err
		}
		doc["name"] = env
	}
	if cnp != "" {
		cnpHash, err := medcrypto.HashCNP(r.master, cnp)
		if err != nil {
			return primitive.NilObjectID, err
		}
		env, err := medcrypto.EncryptString(r.master, "patients.cnp", cnp)
		if err != nil {
			return primitive.NilObjectID, err
		}
		doc["cnp_hash"] = cnpHash
		doc["cnp"] = env
		filter["cnp_hash"] = cnpHash
	} else {
		filter["_id"] = primitive.NewObjectID()
	}

	patients := r.db.Collection("patients")
	update := bson.M{"$setOnInsert": doc}
	res, err := patients.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return primitive.NilObjectID, err
	}
	if oid, ok := res.UpsertedID.(primitive.ObjectID); ok {
		return oid, nil
	}

	var out struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := patients.FindOne(ctx, filter).Decode(&out); err != nil {
		return primitive.NilObjectID, err
	}
	return out.ID, nil
}
