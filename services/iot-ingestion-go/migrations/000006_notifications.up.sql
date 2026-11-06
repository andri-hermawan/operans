-- C5 (ADR 0009): antrian notifikasi (outbox). Setiap kiriman WhatsApp dicatat: menunggu / terkirim / gagal,
-- jumlah percobaan & pesan galat. Pengirim terpisah mencoba ulang dengan jeda bertahap, sehingga server WA
-- yang mati / lambat tidak pernah menghambat penyimpanan telemetry & alert.
CREATE TABLE iot.notifications (
    id                 bigint        GENERATED ALWAYS AS IDENTITY,
    alert_id           bigint        NOT NULL,
    project_id         uuid,
    channel            varchar(20)   NOT NULL DEFAULT 'whatsapp',
    recipient_user_id  uuid,                         -- auth.users (core); tanpa FK lintas schema
    phone              varchar(20)   NOT NULL,       -- dinormalkan: 62xxxxxxxx
    message            text          NOT NULL,
    status             varchar(10)   NOT NULL DEFAULT 'pending',
    attempts           smallint      NOT NULL DEFAULT 0,
    last_error         varchar(500),
    next_attempt_at    timestamptz   NOT NULL DEFAULT now(),   -- juga "sewa" saat sedang dikirim
    sent_at            timestamptz,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    updated_at         timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_notifications          PRIMARY KEY (id),
    CONSTRAINT fk_notifications_alert    FOREIGN KEY (alert_id) REFERENCES iot.alerts (id),
    CONSTRAINT uq_notifications_target   UNIQUE (alert_id, channel, phone),   -- satu nomor satu pesan per alert
    CONSTRAINT ck_notifications_channel  CHECK (channel IN ('whatsapp')),
    CONSTRAINT ck_notifications_status   CHECK (status IN ('pending', 'sent', 'failed'))
);
CREATE INDEX idx_notifications_due     ON iot.notifications (next_attempt_at) WHERE status = 'pending';
CREATE INDEX idx_notifications_project ON iot.notifications (project_id, created_at DESC);

GRANT SELECT, INSERT, UPDATE ON iot.notifications TO iot_ingest;
