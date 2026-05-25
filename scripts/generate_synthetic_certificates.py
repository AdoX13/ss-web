#!/usr/bin/env python3
import json
import os
import random
import time
from datetime import datetime, timedelta
from PIL import Image, ImageDraw, ImageFont, ImageFilter

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
OUTPUT_DIR = os.path.join(PROJECT_ROOT, "medical-images", "synthetic")

# Lists for random generation
NAMES = ["Popescu", "Ionescu", "Dumitru", "Stoica", "Radu", "Gheorghe", "Matei", "Florea", "Costea", "Marinescu"]
PRENUME = ["Ion", "Maria", "Andrei", "Elena", "Radu", "Ana", "George", "Ioana", "Mihai", "Cristina"]
PROFESSIONS = ["Inginer", "Programator", "Medic", "Profesor", "Contabil", "Sofer", "Manager", "Student", "Asistent", "Operator"]
CITIES = ["Bucuresti", "Cluj", "Timisoara", "Iasi", "Constanta", "Craiova", "Brasov", "Galati", "Ploiesti", "Oradea"]
MEDICAL_UNITS = ["Regina Maria", "MedLife", "Sanador", "Medicover", "Synevo", "Clinica Sante"]
COMPANIES = ["Tech SRL", "Soft SA", "Construct SRL", "Auto SRL", "Consulting SA", "Logistics SRL"]

def generate_cnp(year_offset=30):
    year = datetime.now().year - year_offset
    gender = random.choice([1, 2])
    yy = str(year)[-2:]
    mm = f"{random.randint(1, 12):02d}"
    dd = f"{random.randint(1, 28):02d}"
    county = f"{random.randint(1, 52):02d}"
    seq = f"{random.randint(1, 999):03d}"
    # Random check digit
    check = str(random.randint(0, 9))
    return f"{gender}{yy}{mm}{dd}{county}{seq}{check}"

def generate_certificate_data(index):
    exam_date = datetime.now() - timedelta(days=random.randint(0, 365))
    next_exam_date = exam_date + timedelta(days=365)
    
    nume = random.choice(NAMES)
    prenume = random.choice(PRENUME)
    cnp = generate_cnp(random.randint(20, 60))
    profession = random.choice(PROFESSIONS)
    workplace = random.choice(CITIES)
    
    # Weight medical opinions towards APT (70%)
    aviz_types = ["APT", "APT CONDIȚIONAT", "INAPT TEMPORAR", "INAPT"]
    aviz = random.choices(aviz_types, weights=[70, 15, 10, 5], k=1)[0]
    
    control_types = ["Angajare", "Control periodic", "Adaptare", "Reluare", "Supraveghere", "Alte"]
    control = random.choice(control_types)
    
    recomandari = "Fara" if aviz == "APT" else "Reevaluare in 30 de zile"
    
    return {
        "id": f"cert_{index:03d}",
        "unitate_medicala": random.choice(MEDICAL_UNITS),
        "adresa_unitate": f"Str. Principala nr. {random.randint(1, 100)}, {random.choice(CITIES)}",
        "telefon_unitate": f"021{random.randint(1000000, 9999999)}",
        "numar_fisa": str(random.randint(1000, 9999)),
        "societate": random.choice(COMPANIES),
        "adresa_angajator": f"Str. Muncii nr. {random.randint(1, 100)}, {random.choice(CITIES)}",
        "telefon_angajator": f"021{random.randint(1000000, 9999999)}",
        "nume": nume,
        "prenume": prenume,
        "cnp": cnp,
        "profesie": profession,
        "loc_munca": workplace,
        "tip_control": control,
        "aviz": aviz,
        "recomandari": recomandari,
        "data_examen": exam_date.strftime("%d/%m/%Y"),
        "data_urm_examen": next_exam_date.strftime("%d/%m/%Y"),
        "medic": "Dr. " + random.choice(NAMES)
    }

def draw_text(draw, pos, text, font, fill="black"):
    draw.text(pos, text, fill=fill, font=font)

def draw_checkbox(draw, pos, size, checked):
    x, y = pos
    draw.rectangle([x, y, x + size, y + size], outline="black", width=2)
    if checked:
        draw.line([x, y, x + size, y + size], fill="black", width=2)
        draw.line([x, y + size, x + size, y], fill="black", width=2)

def generate_image(data, output_path, add_noise=False):
    width, height = 800, 1100
    img = Image.new("RGB", (width, height), "white")
    draw = ImageDraw.Draw(img)
    
    # Try to load a generic font, fallback to default
    try:
        font_large = ImageFont.truetype("Arial", 24)
        font_medium = ImageFont.truetype("Arial", 18)
        font_small = ImageFont.truetype("Arial", 14)
    except IOError:
        font_large = ImageFont.load_default()
        font_medium = ImageFont.load_default()
        font_small = ImageFont.load_default()

    margin = 50
    y = margin

    # Header
    draw_text(draw, (width//2 - 100, y), "FIȘA DE APTITUDINE", font=font_large)
    y += 40
    
    # Medical Unit
    draw_text(draw, (margin, y), f"Unitate Medicală: {data['unitate_medicala']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Adresa: {data['adresa_unitate']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Telefon: {data['telefon_unitate']}", font=font_medium)
    y += 40
    
    # Employer
    draw_text(draw, (margin, y), f"Societate/Unitate: {data['societate']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Adresa Angajator: {data['adresa_angajator']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Telefon Angajator: {data['telefon_angajator']}", font=font_medium)
    y += 40

    # Personal info
    draw_text(draw, (margin, y), f"Nume: {data['nume']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Prenume: {data['prenume']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"CNP: {data['cnp']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Profesie/Funcție: {data['profesie']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Loc de muncă: {data['loc_munca']}", font=font_medium)
    y += 40

    # Tip Control
    draw_text(draw, (margin, y), "Tip Control:", font=font_medium)
    y += 30
    controls = ["Angajare", "Control periodic", "Adaptare", "Reluare", "Supraveghere", "Alte"]
    for i, c in enumerate(controls):
        x = margin + (i % 3) * 200
        y_offset = y + (i // 3) * 30
        draw_checkbox(draw, (x, y_offset), 15, data['tip_control'] == c)
        draw_text(draw, (x + 25, y_offset), c, font=font_medium)
    y += 70

    # Aviz Medical
    draw_text(draw, (margin, y), "Aviz Medical:", font=font_medium)
    y += 30
    avize = ["APT", "APT CONDIȚIONAT", "INAPT TEMPORAR", "INAPT"]
    for i, a in enumerate(avize):
        x = margin + (i % 2) * 300
        y_offset = y + (i // 2) * 30
        draw_checkbox(draw, (x, y_offset), 15, data['aviz'] == a)
        draw_text(draw, (x + 25, y_offset), a, font=font_medium)
    y += 70

    # Recomandari & Dates
    draw_text(draw, (margin, y), f"Recomandări: {data['recomandari']}", font=font_medium)
    y += 40
    draw_text(draw, (margin, y), f"Data: {data['data_examen']}", font=font_medium)
    y += 25
    draw_text(draw, (margin, y), f"Data următoarei examinări: {data['data_urm_examen']}", font=font_medium)
    y += 60
    
    # Signature
    draw_text(draw, (width - 250, y), "Semnătură medic:", font=font_medium)
    y += 25
    draw_text(draw, (width - 250, y), data['medic'], font=font_medium)

    # Add slight rotation and noise for realism
    if add_noise:
        angle = random.uniform(-2, 2)
        img = img.rotate(angle, resample=Image.BICUBIC, fillcolor="white")
        
        # Add blur
        img = img.filter(ImageFilter.GaussianBlur(radius=0.5))

    img.save(output_path, "JPEG", quality=85)

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    manifest = []
    
    print(f"Generating synthetic medical certificates in {OUTPUT_DIR}...")
    for i in range(1, 13):
        data = generate_certificate_data(i)
        
        # Alternate between clean and noisy
        add_noise = i % 2 == 0
        filename = f"{data['id']}_{'noisy' if add_noise else 'clean'}.jpg"
        output_path = os.path.join(OUTPUT_DIR, filename)
        
        generate_image(data, output_path, add_noise=add_noise)
        
        data['filename'] = filename
        manifest.append(data)
        print(f"  Generated {filename}")
        
    manifest_path = os.path.join(OUTPUT_DIR, "manifest.json")
    with open(manifest_path, 'w', encoding='utf-8') as f:
        json.dump(manifest, f, ensure_ascii=False, indent=2)
    
    print(f"✅ Generated {len(manifest)} synthetic certificates.")
    print(f"   Manifest saved to: {manifest_path}")

if __name__ == "__main__":
    main()
