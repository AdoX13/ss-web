package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureSchema installs collection validators and indexes used by P6 reports,
// audit, evidence, and encrypted PHI lookups. It is intentionally best-effort:
// existing developer data should not prevent the service from starting.
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	collections := map[string]bson.M{
		"photos":          photosValidator(),
		"users":           usersValidator(),
		"devices":         devicesValidator(),
		"review_items":    reviewItemsValidator(),
		"audit_log":       auditLogValidator(),
		"evidence_chain":  evidenceChainValidator(),
		"patients":        patientsValidator(),
		"medical_records": medicalRecordsValidator(),
		"ocr_results":     ocrResultsValidator(),
	}
	for name, validator := range collections {
		if err := ensureCollection(ctx, db, name, validator); err != nil {
			return err
		}
	}
	return ensureIndexes(ctx, db)
}

func ensureCollection(ctx context.Context, db *mongo.Database, name string, validator bson.M) error {
	err := db.CreateCollection(ctx, name, options.CreateCollection().
		SetValidator(validator).
		SetValidationLevel("moderate"))
	if err == nil {
		return nil
	}
	if !isNamespaceExists(err) {
		return err
	}
	cmd := bson.D{
		{Key: "collMod", Value: name},
		{Key: "validator", Value: validator},
		{Key: "validationLevel", Value: "moderate"},
	}
	return db.RunCommand(ctx, cmd).Err()
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		"photos": {
			{Keys: bson.D{{Key: "timestamp", Value: -1}}},
			{Keys: bson.D{{Key: "data", Value: -1}}},
			{Keys: bson.D{{Key: "data_urm_examinari", Value: 1}}},
			{Keys: bson.D{{Key: "device_id", Value: 1}}},
		},
		"users": {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		"review_items": {
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "original_confidence", Value: 1}}},
			{Keys: bson.D{{Key: "created_at", Value: -1}}},
			{Keys: bson.D{{Key: "image_id", Value: 1}}},
		},
		"audit_log": {
			{Keys: bson.D{{Key: "ts", Value: -1}}},
			{Keys: bson.D{{Key: "actor_email", Value: 1}, {Key: "ts", Value: -1}}},
			{Keys: bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}}},
		},
		"evidence_chain": {
			{Keys: bson.D{{Key: "seq", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "this_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		"patients": {
			{Keys: bson.D{{Key: "cnp_hash", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		},
		"medical_records": {
			{Keys: bson.D{{Key: "exam_date", Value: -1}}},
			{Keys: bson.D{{Key: "expiration_date", Value: 1}}},
			{Keys: bson.D{{Key: "patient_id", Value: 1}}},
		},
		"ocr_results": {
			{Keys: bson.D{{Key: "document_id", Value: 1}}},
			{Keys: bson.D{{Key: "extracted_at", Value: -1}}},
		},
	}
	for collection, models := range indexes {
		if len(models) == 0 {
			continue
		}
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}
	return nil
}

func isNamespaceExists(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) && commandError.Code == 48
}

func schema(required []string, props bson.M) bson.M {
	return bson.M{"$jsonSchema": bson.M{
		"bsonType":             "object",
		"required":             required,
		"additionalProperties": true,
		"properties":           props,
	}}
}

func photosValidator() bson.M {
	return schema([]string{"timestamp", "image_type", "device_id"}, bson.M{
		"timestamp": bson.M{"bsonType": "date"},
		"image_type": bson.M{"bsonType": "string"},
		"device_id": bson.M{"bsonType": "string"},
		"cnp": bson.M{"bsonType": "string"},
		"overall_confidence": bson.M{"bsonType": []string{"double", "int", "long", "decimal", "null"}},
		"needs_review": bson.M{"bsonType": []string{"bool", "null"}},
	})
}

func usersValidator() bson.M {
	return schema([]string{"email", "password", "role", "active"}, bson.M{
		"email": bson.M{"bsonType": "string"},
		"password": bson.M{"bsonType": "string"},
		"role": bson.M{"enum": bson.A{"admin", "doctor", "researcher", "auditor"}},
		"active": bson.M{"bsonType": "bool"},
	})
}

func devicesValidator() bson.M {
	return schema([]string{"device_id", "device_name", "device_status"}, bson.M{
		"device_id": bson.M{"bsonType": "string"},
		"device_name": bson.M{"bsonType": "string"},
		"device_status": bson.M{"bsonType": "string"},
	})
}

func reviewItemsValidator() bson.M {
	return schema([]string{"image_id", "field_name", "original_confidence", "status", "created_at"}, bson.M{
		"image_id": bson.M{"bsonType": "objectId"},
		"field_name": bson.M{"bsonType": "string"},
		"original_confidence": bson.M{"bsonType": []string{"double", "int", "long", "decimal"}},
		"status": bson.M{"enum": bson.A{"pending", "approved", "corrected", "rejected"}},
		"created_at": bson.M{"bsonType": "date"},
	})
}

func auditLogValidator() bson.M {
	return schema([]string{"ts", "actor_email", "action", "resource_type"}, bson.M{
		"ts": bson.M{"bsonType": "date"},
		"actor_email": bson.M{"bsonType": "string"},
		"actor_ip": bson.M{"bsonType": "string"},
		"action": bson.M{"bsonType": "string"},
		"resource_type": bson.M{"bsonType": "string"},
		"resource_id": bson.M{"bsonType": "string"},
		"details": bson.M{"bsonType": "object"},
	})
}

func evidenceChainValidator() bson.M {
	return schema([]string{"seq", "prev_hash", "this_hash", "signature", "payload", "created_at"}, bson.M{
		"seq": bson.M{"bsonType": []string{"int", "long"}},
		"prev_hash": bson.M{"bsonType": "string"},
		"this_hash": bson.M{"bsonType": "string"},
		"signature": bson.M{"bsonType": "string"},
		"payload": bson.M{"bsonType": "object"},
		"created_at": bson.M{"bsonType": "date"},
	})
}

func patientsValidator() bson.M {
	return schema([]string{"created_at"}, bson.M{
		"name": bson.M{"bsonType": "object"},
		"cnp_hash": bson.M{"bsonType": "string"},
		"cnp": bson.M{"bsonType": "object"},
		"dob": bson.M{"bsonType": "object"},
		"created_at": bson.M{"bsonType": "date"},
	})
}

func medicalRecordsValidator() bson.M {
	return schema([]string{"created_at"}, bson.M{
		"patient_id": bson.M{"bsonType": "objectId"},
		"image_id": bson.M{"bsonType": "objectId"},
		"control_type": bson.M{"bsonType": "string"},
		"medical_opinion": bson.M{"bsonType": "string"},
		"exam_date": bson.M{"bsonType": "date"},
		"expiration_date": bson.M{"bsonType": "date"},
		"doctor_name": bson.M{"bsonType": "object"},
		"workplace": bson.M{"bsonType": "object"},
		"profession": bson.M{"bsonType": "string"},
		"created_at": bson.M{"bsonType": "date"},
	})
}

func ocrResultsValidator() bson.M {
	return schema([]string{"document_id", "extracted_at", "overall_confidence", "needs_review"}, bson.M{
		"document_id": bson.M{"bsonType": "string"},
		"extracted_at": bson.M{"bsonType": "date"},
		"overall_confidence": bson.M{"bsonType": []string{"double", "int", "long", "decimal"}},
		"needs_review": bson.M{"bsonType": "bool"},
		"processing_ms": bson.M{"bsonType": []string{"int", "long", "null"}},
	})
}
