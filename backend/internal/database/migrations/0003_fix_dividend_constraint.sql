ALTER TABLE asset_event DROP CONSTRAINT IF EXISTS asset_event_asset_id_ex_date_type_key;

ALTER TABLE asset_event ADD CONSTRAINT asset_event_payment_profile_key UNIQUE NULLS NOT DISTINCT (asset_id, type, gross_amount, payment_date);
