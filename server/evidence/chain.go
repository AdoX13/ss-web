// Package evidence provides the hash-chained, Ed25519-signed evidence plane.
package evidence

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	medcrypto "mqtt-streaming-server/crypto"
)

// Entry is one evidence plane record.
type Entry struct {
	ActorEmail string
	Action     string
	Payload    map[string]any
}

// Chain appends tamper-evident entries. Use Noop when a concrete no-op chain
// is desired for tests or degraded local startup.
type Chain interface {
	Append(ctx context.Context, e Entry) error
}

type Verifier interface {
	Verify(ctx context.Context) (*Verification, error)
}

type Verification struct {
	Valid    bool   `json:"valid"`
	Records  int64  `json:"records"`
	LastHash string `json:"last_hash,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Record struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Seq        int64              `json:"seq" bson:"seq"`
	PrevHash   string             `json:"prev_hash" bson:"prev_hash"`
	ThisHash   string             `json:"this_hash" bson:"this_hash"`
	Signature  string             `json:"signature" bson:"signature"`
	ActorEmail string             `json:"actor_email" bson:"actor_email"`
	Action     string             `json:"action" bson:"action"`
	Payload    map[string]any     `json:"payload" bson:"payload"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}

type MongoChain struct {
	collection *mongo.Collection
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	now        func() time.Time
}

func NewMongoChain(db *mongo.Database, privateKey ed25519.PrivateKey) (*MongoChain, error) {
	publicKey, err := medcrypto.PublicKeyFromPrivate(privateKey)
	if err != nil {
		return nil, err
	}
	return &MongoChain{
		collection: db.Collection("evidence_chain"),
		privateKey: privateKey,
		publicKey:  publicKey,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

func (m *MongoChain) Append(ctx context.Context, e Entry) error {
	if e.Action == "" {
		return errors.New("evidence: action is required")
	}
	if e.Payload == nil {
		e.Payload = map[string]any{}
	}

	latest, err := m.latest(ctx)
	if err != nil {
		return err
	}
	rec := Record{
		ID:         primitive.NewObjectID(),
		Seq:        latest.Seq + 1,
		PrevHash:   latest.ThisHash,
		ActorEmail: e.ActorEmail,
		Action:     e.Action,
		Payload:    e.Payload,
		CreatedAt:  m.now(),
	}
	hash, err := recordHash(rec)
	if err != nil {
		return err
	}
	rec.ThisHash = hash
	sig, err := medcrypto.SignEd25519(m.privateKey, []byte(hash))
	if err != nil {
		return err
	}
	rec.Signature = base64.StdEncoding.EncodeToString(sig)

	_, err = m.collection.InsertOne(ctx, rec)
	return err
}

func (m *MongoChain) Verify(ctx context.Context) (*Verification, error) {
	cursor, err := m.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "seq", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var prev string
	var count int64
	var expectedSeq int64 = 1
	for cursor.Next(ctx) {
		var rec Record
		if err := cursor.Decode(&rec); err != nil {
			return nil, err
		}
		if rec.Seq != expectedSeq {
			return invalid(count, fmt.Sprintf("expected seq %d, got %d", expectedSeq, rec.Seq)), nil
		}
		if rec.PrevHash != prev {
			return invalid(count, fmt.Sprintf("seq %d prev_hash mismatch", rec.Seq)), nil
		}
		hash, err := recordHash(rec)
		if err != nil {
			return nil, err
		}
		if hash != rec.ThisHash {
			return invalid(count, fmt.Sprintf("seq %d hash mismatch", rec.Seq)), nil
		}
		sig, err := base64.StdEncoding.DecodeString(rec.Signature)
		if err != nil {
			return invalid(count, fmt.Sprintf("seq %d invalid signature encoding", rec.Seq)), nil
		}
		if !medcrypto.VerifyEd25519(m.publicKey, []byte(rec.ThisHash), sig) {
			return invalid(count, fmt.Sprintf("seq %d signature verification failed", rec.Seq)), nil
		}

		prev = rec.ThisHash
		count++
		expectedSeq++
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return &Verification{Valid: true, Records: count, LastHash: prev}, nil
}

func (m *MongoChain) latest(ctx context.Context) (Record, error) {
	var rec Record
	err := m.collection.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "seq", Value: -1}})).Decode(&rec)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Record{}, nil
	}
	return rec, err
}

func invalid(records int64, msg string) *Verification {
	return &Verification{Valid: false, Records: records, Error: msg}
}

type canonicalRecord struct {
	Seq        int64          `json:"seq"`
	PrevHash   string         `json:"prev_hash"`
	ActorEmail string         `json:"actor_email"`
	Action     string         `json:"action"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  string         `json:"created_at"`
}

func recordHash(rec Record) (string, error) {
	payload := canonicalRecord{
		Seq:        rec.Seq,
		PrevHash:   rec.PrevHash,
		ActorEmail: rec.ActorEmail,
		Action:     rec.Action,
		Payload:    rec.Payload,
		CreatedAt:  rec.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Noop discards every entry. Use in tests or degraded local startup only.
type Noop struct{}

func (n *Noop) Append(_ context.Context, _ Entry) error { return nil }

func (n *Noop) Verify(_ context.Context) (*Verification, error) {
	return &Verification{Valid: true}, nil
}
