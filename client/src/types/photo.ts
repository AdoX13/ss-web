// Photo type as exposed to the frontend (backend handoff §3.1).
// The CNP field is PHI: do NOT render it for non-admin roles.

export interface Photo {
  id: string;
  timestamp: string; // ISO 8601
  image_type: string; // "jpeg" | "png"
  presigned_url?: string;
  device_id: string;
  text: string; // raw OCR text

  // Medical fields (populated by the OCR worker)
  unitate_medicala?: string;
  numar_fisa?: string;
  societate_unitate?: string;
  nume?: string;
  prenume?: string;
  cnp?: string; // PHI — see backend handoff §8
  profesie_functie?: string;
  loc_de_munca?: string;
  tip_control?: string;
  aviz_medical?: string;
  data?: string;
  data_urm_examinari?: string;

  // OCR quality metadata
  overall_confidence: number; // 0.0 – 1.0; below 0.95 → needs_review
  needs_review: boolean;
}
