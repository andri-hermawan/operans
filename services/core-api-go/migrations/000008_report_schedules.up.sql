-- Fase D/F5: jadwal kirim laporan otomatis (pengganti report_schedules sistem lama).
-- Laporan dirender sebagai PDF dari halaman cetak frontend lalu dikirim email ke penerima.

CREATE TABLE core.report_schedules (
    id                  uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id          uuid          NOT NULL,
    name                varchar(150)  NOT NULL,
    report_type         varchar(50)   NOT NULL,             -- alert_summary
    shift_code          varchar(20),                         -- NULL = satu hari operasional penuh
    date_offset         smallint      NOT NULL DEFAULT -1,  -- -1 = data kemarin (H-1), 0 = hari ini
    send_time           time          NOT NULL DEFAULT '09:00',  -- jam lokal project
    recipient_user_ids  uuid[]        NOT NULL DEFAULT '{}',
    is_active           boolean       NOT NULL DEFAULT true,
    last_run_at         timestamptz,
    created_at          timestamptz   NOT NULL DEFAULT now(),
    created_by          uuid,
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    updated_by          uuid,

    CONSTRAINT pk_report_schedules          PRIMARY KEY (id),
    CONSTRAINT fk_report_schedules_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT ck_report_schedules_offset   CHECK (date_offset IN (-1, 0)),
    CONSTRAINT ck_report_schedules_type     CHECK (report_type IN ('alert_summary'))
);
CREATE INDEX idx_report_schedules_active ON core.report_schedules (is_active, send_time);

CREATE TABLE core.report_schedule_logs (
    id             bigint        GENERATED ALWAYS AS IDENTITY,
    schedule_id    uuid          NOT NULL,
    run_date       date          NOT NULL,   -- tanggal kirim (jam lokal project)
    report_date    date          NOT NULL,   -- tanggal data yang dilaporkan
    trigger        varchar(20)   NOT NULL,   -- schedule | manual
    status         varchar(20)   NOT NULL,   -- sent | failed
    recipients     integer       NOT NULL DEFAULT 0,
    error_message  text,
    created_at     timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_report_schedule_logs           PRIMARY KEY (id),
    CONSTRAINT fk_report_schedule_logs_schedule  FOREIGN KEY (schedule_id) REFERENCES core.report_schedules (id) ON DELETE CASCADE,
    CONSTRAINT ck_report_schedule_logs_trigger   CHECK (trigger IN ('schedule', 'manual')),
    CONSTRAINT ck_report_schedule_logs_status    CHECK (status IN ('sent', 'failed'))
);
CREATE INDEX idx_report_schedule_logs_schedule ON core.report_schedule_logs (schedule_id, run_date);

CREATE TRIGGER trg_report_schedules_updated_at BEFORE UPDATE ON core.report_schedules
    FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.report_schedules     TO core_api;
GRANT SELECT, INSERT                 ON core.report_schedule_logs TO core_api;
