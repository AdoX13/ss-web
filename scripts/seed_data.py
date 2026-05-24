import random
import base64
import binascii
import hashlib
import hmac
import os
import secrets
from datetime import datetime, timedelta
import pymongo
from bson import ObjectId

try:
    from cryptography.hazmat.primitives.ciphers.aead import AESGCM
except ImportError:
    AESGCM = None

# Database Connection
MONGO_USER = os.environ.get("MONGO_INITDB_ROOT_USERNAME", "admin")
MONGO_PASSWORD = os.environ.get("MONGO_INITDB_ROOT_PASSWORD", "supersecret")
MONGO_URI = os.environ.get(
    "MONGO_URI",
    f"mongodb://{MONGO_USER}:{MONGO_PASSWORD}@localhost:27019/?authSource=admin",
)
DB_NAME = "mqtt-streaming-server"
COLLECTION_NAME = "photos"
ALG = "AES-256-GCM-HKDF-SHA256"
ENVELOPE_VERSION = 1

# Fake Data Sources
NAMES = ["Ion", "Maria", "Andrei", "Elena", "Radu", "Ana", "George", "Ioana", "Mihai", "Cristina", "Alexandru", "Gabriela", "Florin", "Daniela", "Vlad"]
SURNAMES = ["Popescu", "Ionescu", "Dumitru", "Stoica", "Radu", "Gheorghe", "Matei", "Florea", "Costea", "Marinescu", "Dinu", "Toma", "Stanciu", "Neagu", "Preda"]
JOBS = ["Inginer", "Programator", "Medic", "Profesor", "Contabil", "Sofer", "Manager", "Student", "Asistent", "Operator"]
DOCTORS = ["Dr. Andrei Marin", "Dr. Elena Dobre", "Dr. Ioana Pavel"]

def generate_random_photo():
    timestamp = datetime.now() - timedelta(days=random.randint(0, 30))
    nume = random.choice(SURNAMES)
    prenume = random.choice(NAMES)
    confidence = round(random.uniform(0.72, 0.99), 3)
    
    control_types = ["Angajare", "Periodic", "Adaptare", "Reluare", "Supraveghere", "Alte"]
    selected_control = random.choice(control_types)
    
    control_angajare = selected_control == "Angajare"
    control_periodic = selected_control == "Periodic"
    control_adaptare = selected_control == "Adaptare"
    control_reluare = selected_control == "Reluare"
    control_supraveghere = selected_control == "Supraveghere"
    control_alte = selected_control == "Alte"

    aviz_types = ["APT", "APT Conditionat", "Inapt Temporar", "Inapt"]
    selected_aviz = random.choices(aviz_types, weights=[70, 15, 10, 5], k=1)[0]
    
    aviz_apt = selected_aviz == "APT"
    aviz_apt_conditionat = selected_aviz == "APT Conditionat"
    aviz_inapt_temporar = selected_aviz == "Inapt Temporar"
    aviz_inapt = selected_aviz == "Inapt"

    return {
        "_id": ObjectId(),
        "timestamp": timestamp,
        "image_type": "jpeg",
        "device_id": f"device-{random.randint(1, 5)}",
        "text": f"Fake OCR for {nume} {prenume}",
        
        "unitate_medicala": "Clinica Test",
        "adresa_unitate_medicala": "Str. Testului nr 1",
        "telefon_unitate_medicala": "0700000000",
        "numar_fisa": f"FISA-{random.randint(1000, 9999)}",
        "societate_unitate": "Compania SRL",
        "adresa_angajator": "Bd. Muncii nr 10",
        "telefon_angajator": "0711111111",
        "nume": nume,
        "prenume": prenume,
        "cnp": f"{random.randint(1, 2)}{random.randint(50, 99)}{random.randint(10, 12)}{random.randint(10, 28)}123456",
        "profesie_functie": random.choice(JOBS),
        "loc_de_munca": "Bucuresti",
        "doctor_name": random.choice(DOCTORS),
        
        "tip_control": f"Control {selected_control}",
        "control_angajare": control_angajare,
        "control_periodic": control_periodic,
        "control_adaptare": control_adaptare,
        "control_reluare": control_reluare,
        "control_supraveghere": control_supraveghere,
        "control_alte": control_alte,
        
        "aviz_medical": selected_aviz,
        "aviz_apt": aviz_apt,
        "aviz_apt_conditionat": aviz_apt_conditionat,
        "aviz_inapt_temporar": aviz_inapt_temporar,
        "aviz_inapt": aviz_inapt,
        
        "recomandari": "Nicio recomandare" if aviz_apt else "Reevaluare in 30 zile",
        "data": timestamp,
        "data_urm_examinari": timestamp + timedelta(days=365),
        "overall_confidence": confidence,
        "needs_review": confidence < 0.82,
    }

def parse_master_key(value):
    value = (value or "").strip()
    if not value:
        raise ValueError("MEDSEC_MASTER_KEY is not set")

    decoders = [
        lambda s: base64.b64decode(s, validate=True),
        lambda s: base64.b64decode(s + "=" * (-len(s) % 4), validate=True),
        lambda s: binascii.unhexlify(s),
        lambda s: s.encode("utf-8"),
    ]
    for decode in decoders:
        try:
            key = decode(value)
        except (binascii.Error, ValueError):
            continue
        if len(key) == 32:
            return key
    raise ValueError("MEDSEC_MASTER_KEY must decode to exactly 32 bytes")

def hkdf_sha256(secret, salt, info, size=32):
    if not salt:
        salt = bytes(hashlib.sha256().digest_size)
    prk = hmac.new(salt, secret, hashlib.sha256).digest()
    out = b""
    previous = b""
    counter = 1
    while len(out) < size:
        previous = hmac.new(prk, previous + info + bytes([counter]), hashlib.sha256).digest()
        out += previous
        counter += 1
    return out[:size]

def derive_field_key(master, field):
    return hkdf_sha256(
        master,
        b"medsec-ocr-field-encryption-v1",
        field.encode("utf-8"),
        32,
    )

def encrypt_string(master, field, plaintext):
    if AESGCM is None:
        raise RuntimeError("cryptography is required for encrypted seed data")
    nonce = secrets.token_bytes(12)
    key = derive_field_key(master, field)
    aad = f"{ALG}:{ENVELOPE_VERSION}:{field}".encode("utf-8")
    ciphertext = AESGCM(key).encrypt(nonce, plaintext.encode("utf-8"), aad)
    return {
        "alg": ALG,
        "v": ENVELOPE_VERSION,
        "field": field,
        "nonce": nonce,
        "ciphertext": ciphertext,
    }

def normalize_cnp(cnp):
    return "".join(ch for ch in cnp if ch.isdigit())

def hash_cnp(master, cnp):
    normalized = normalize_cnp(cnp)
    key = derive_field_key(master, "cnp_lookup_hmac")
    return hmac.new(key, normalized.encode("utf-8"), hashlib.sha256).hexdigest()

def seed_encrypted_projection(db, records):
    master_value = os.environ.get("MEDSEC_MASTER_KEY")
    if not master_value:
        print("MEDSEC_MASTER_KEY missing; skipped encrypted patients/medical_records seed.")
        return
    if AESGCM is None:
        print("Python package 'cryptography' missing; skipped encrypted seed projection.")
        print("Install with: pip install cryptography")
        return

    master = parse_master_key(master_value)
    patients = db["patients"]
    medical_records = db["medical_records"]
    ocr_results = db["ocr_results"]
    review_items = db["review_items"]
    now = datetime.utcnow()

    medical_docs = []
    ocr_docs = []
    review_docs = []
    for photo in records:
        name = f"{photo.get('nume', '')} {photo.get('prenume', '')}".strip()
        cnp = normalize_cnp(photo.get("cnp", ""))
        patient_id = None
        if name or cnp:
            patient_doc = {"created_at": now}
            patient_filter = {}
            if name:
                patient_doc["name"] = encrypt_string(master, "patients.name", name)
            if cnp:
                cnp_hash = hash_cnp(master, cnp)
                patient_doc["cnp_hash"] = cnp_hash
                patient_doc["cnp"] = encrypt_string(master, "patients.cnp", cnp)
                patient_filter["cnp_hash"] = cnp_hash
            else:
                patient_id = ObjectId()
                patient_filter["_id"] = patient_id

            result = patients.update_one(
                patient_filter,
                {"$setOnInsert": patient_doc},
                upsert=True,
            )
            patient_id = result.upserted_id or patients.find_one(patient_filter, {"_id": 1})["_id"]

        record = {
            "image_id": photo["_id"],
            "control_type": photo["tip_control"],
            "medical_opinion": photo["aviz_medical"],
            "profession": photo["profesie_functie"],
            "exam_date": photo["data"],
            "expiration_date": photo["data_urm_examinari"],
            "created_at": now,
        }
        if patient_id:
            record["patient_id"] = patient_id
        if photo.get("loc_de_munca"):
            record["workplace"] = encrypt_string(master, "medical_records.workplace", photo["loc_de_munca"])
        if photo.get("doctor_name"):
            record["doctor_name"] = encrypt_string(master, "medical_records.doctor_name", photo["doctor_name"])
        medical_docs.append(record)

        ocr_docs.append({
            "document_id": str(photo["_id"]),
            "extracted_at": photo["timestamp"],
            "overall_confidence": photo["overall_confidence"],
            "needs_review": photo["needs_review"],
            "processing_ms": random.randint(250, 2500),
        })
        if photo["needs_review"]:
            review_docs.append({
                "image_id": photo["_id"],
                "field_name": "patient_name",
                "original_value": name,
                "original_confidence": photo["overall_confidence"],
                "status": "pending",
                "created_at": now,
            })

    if medical_docs:
        medical_records.insert_many(medical_docs)
    if ocr_docs:
        ocr_results.insert_many(ocr_docs)
    if review_docs:
        review_items.insert_many(review_docs)
    print(f"Inserted encrypted projection: {len(medical_docs)} medical records, {len(ocr_docs)} OCR rows, {len(review_docs)} review items.")

def seed_data():
    try:
        client = pymongo.MongoClient(MONGO_URI)
        db = client[DB_NAME]
        
        print("Seeding Users...")
        users_col = db["users"]
        users_col.delete_many({})
        users_col.insert_many(get_users())
        print(f"Inserted {users_col.count_documents({})} users.")

        print("Seeding Photos/Records...")
        photos_col = db["photos"]
        photos_col.delete_many({})
        records = [generate_random_photo() for _ in range(count)]
        photos_col.insert_many(records)
        print(f"Inserted {photos_col.count_documents({})} photo records.")
        
        # Patients collection based on photos
        print("Seeding Patients...")
        patients_col = db["patients"]
        patients_col.delete_many({})
        patients = []
        for r in records:
            patients.append({
                "name": f"{r['nume']} {r['prenume']}",
                "cnp": r['cnp'],
                "cnp_hash": str(hash(r['cnp'])),
                "dob": r['cnp'][1:7],  # rough extraction for demo
                "created_at": r['timestamp']
            })
        patients_col.insert_many(patients)
        print(f"Inserted {patients_col.count_documents({})} patients.")

        # Audit log collection
        print("Seeding Audit Log...")
        audit_col = db["audit_log"]
        audit_col.delete_many({})
        audit_logs = []
        for i in range(5):
            audit_logs.append({
                "ts": datetime.now() - timedelta(minutes=random.randint(1, 60)),
                "actor_id": "admin@medsec.ro",
                "actor_ip": "127.0.0.1",
                "action": "READ_REPORT",
                "resource_type": "report",
                "resource_id": "r1",
                "details": "Viewed compliance report"
            })
        audit_col.insert_many(audit_logs)
        print(f"Inserted {audit_col.count_documents({})} audit logs.")
        
        result = collection.insert_many(records)
        print(f"Successfully inserted {len(result.inserted_ids)} records!")
        seed_encrypted_projection(db, records)
        
        new_count = collection.count_documents({})
        print(f"New document count: {new_count}")
        print("✅ Data seeded successfully!")
        
    except Exception as e:
        print(f"An error occurred: {e}")
        print("Ensure 'pymongo' and 'bcrypt' are installed: pip install pymongo bcrypt")

if __name__ == "__main__":
    import sys
    count = 15
    if len(sys.argv) > 1:
        try:
            count = int(sys.argv[1])
        except ValueError:
            pass
    seed_data(count)
