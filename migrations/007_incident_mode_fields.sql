ALTER TABLE incidents ADD COLUMN incident_mode TEXT CHECK (
  incident_mode IS NULL
  OR incident_mode IN ('emergency', 'interaction_record', 'safety_check', 'evidence_note')
);

ALTER TABLE incidents ADD COLUMN capture_profile TEXT CHECK (
  capture_profile IS NULL
  OR capture_profile IN ('audio_video_location', 'audio_location', 'location_checkin', 'note_or_attachment', 'custom')
);

ALTER TABLE incidents ADD COLUMN escalation_policy TEXT CHECK (
  escalation_policy IS NULL
  OR escalation_policy IN ('none', 'trusted_contacts_on_start', 'trusted_contacts_on_missed_checkin', 'urgent_trusted_contact_alert')
);

ALTER TABLE incidents ADD COLUMN sharing_state TEXT CHECK (
  sharing_state IS NULL
  OR sharing_state IN ('private', 'trusted_contact_access', 'public_link_created', 'legal_export_created', 'revoked_or_expired')
);
