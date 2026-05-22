-- Consent Management API Database Schema

-- Main consent table
CREATE TABLE IF NOT EXISTS CONSENT (
  CONSENT_ID            CHAR(36) NOT NULL,
  CREATED_TIME          BIGINT NOT NULL,
  UPDATED_TIME          BIGINT NOT NULL,
  GROUP_ID              VARCHAR(255) NOT NULL,
  CONSENT_TYPE          VARCHAR(64) NOT NULL,
  CURRENT_STATUS        VARCHAR(64) NOT NULL,
  CONSENT_FREQUENCY     INT DEFAULT NULL,
  EXPIRATION_TIME       BIGINT DEFAULT NULL,
  RECURRING_INDICATOR   BOOLEAN DEFAULT NULL,
  DATA_ACCESS_VALIDITY_DURATION BIGINT DEFAULT NULL,
  ORG_ID                VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (CONSENT_ID),
  INDEX idx_group_id (GROUP_ID),
  INDEX idx_consent_type (CONSENT_TYPE),
  INDEX idx_current_status (CURRENT_STATUS),
  INDEX idx_created_time (CREATED_TIME),
  INDEX idx_updated_time (UPDATED_TIME),
  INDEX idx_org_id (ORG_ID)
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Authorization resource table
CREATE TABLE IF NOT EXISTS CONSENT_AUTH_RESOURCE (
  AUTH_ID           CHAR(36) NOT NULL,
  CONSENT_ID        CHAR(36) NOT NULL,
  AUTH_TYPE         VARCHAR(255) NOT NULL,
  USER_ID           VARCHAR(255) DEFAULT NULL,
  AUTH_STATUS       VARCHAR(255) NOT NULL,
  UPDATED_TIME      BIGINT NOT NULL,
  RESOURCES         BLOB DEFAULT NULL,
  ORG_ID            VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (AUTH_ID),
  INDEX idx_consent_id (CONSENT_ID),
  INDEX idx_user_id (USER_ID),
  INDEX idx_auth_status (AUTH_STATUS),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT FK_CONSENT_AUTH_RESOURCE
    FOREIGN KEY (CONSENT_ID)
    REFERENCES CONSENT (CONSENT_ID)
    ON DELETE CASCADE
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Status audit table for tracking consent status changes
CREATE TABLE IF NOT EXISTS CONSENT_STATUS_AUDIT (
  STATUS_AUDIT_ID   CHAR(36) NOT NULL,
  CONSENT_ID        CHAR(36) NOT NULL,
  CURRENT_STATUS    VARCHAR(64) NOT NULL,
  ACTION_TIME       BIGINT NOT NULL,
  REASON            TEXT DEFAULT NULL,
  ACTION_BY         VARCHAR(255) DEFAULT NULL,
  PREVIOUS_STATUS   VARCHAR(64) DEFAULT NULL,
  ORG_ID            VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (STATUS_AUDIT_ID),
  INDEX idx_consent_id (CONSENT_ID),
  INDEX idx_action_time (ACTION_TIME),
  CONSTRAINT FK_CONSENT_STATUS_AUDIT
    FOREIGN KEY (CONSENT_ID)
    REFERENCES CONSENT (CONSENT_ID)
    ON DELETE CASCADE
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Full amendment history table for pre-mutation consent snapshots
CREATE TABLE IF NOT EXISTS CONSENT_HISTORY (
  HISTORY_ID       CHAR(36)     NOT NULL,
  CONSENT_ID       CHAR(36)     NOT NULL,
  ORG_ID           VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  ACTION_TIME      BIGINT       NOT NULL,
  ACTION_BY        VARCHAR(255) DEFAULT NULL,
  REASON           VARCHAR(255) DEFAULT NULL,
  SNAPSHOT         JSON         NOT NULL,

  PRIMARY KEY (HISTORY_ID),
  INDEX idx_history_consent_time (CONSENT_ID, ACTION_TIME),
  INDEX idx_org_id (ORG_ID),

  CONSTRAINT FK_CONSENT_HISTORY
    FOREIGN KEY (CONSENT_ID)
    REFERENCES CONSENT (CONSENT_ID)
    ON DELETE CASCADE
) ENGINE=INNODB ROW_FORMAT=DYNAMIC DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Consent attributes table for key-value pairs
CREATE TABLE IF NOT EXISTS CONSENT_ATTRIBUTE (
  CONSENT_ID        CHAR(36) NOT NULL,
  ATT_KEY           VARCHAR(255) NOT NULL,
  ATT_VALUE         VARCHAR(1024) NOT NULL,
  ORG_ID            VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (CONSENT_ID, ATT_KEY),
  INDEX idx_att_key (ATT_KEY),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT FK_CONSENT_ATTRIBUTE
    FOREIGN KEY (CONSENT_ID)
    REFERENCES CONSENT (CONSENT_ID)
    ON DELETE CASCADE
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Consent element table — each row is a specific version of an element.
-- ID groups all versions of the same element (application-assigned UUID, never reused).
-- NAME+NAMESPACE uniqueness per org is enforced at the application layer;
-- C2 provides a partial DB-level guard at the version granularity.
CREATE TABLE IF NOT EXISTS ELEMENT (
  VERSION_ID     CHAR(36) NOT NULL,
  ID             CHAR(36) NOT NULL,
  NAME           VARCHAR(255) NOT NULL,
  NAMESPACE      VARCHAR(255) NOT NULL DEFAULT 'default',
  TYPE           VARCHAR(64) NOT NULL DEFAULT 'basic',
  VERSION        INT UNSIGNED NOT NULL,
  DISPLAY_NAME   VARCHAR(255) DEFAULT NULL,
  DESCRIPTION    VARCHAR(1024) DEFAULT NULL,
  ELEMENT_SCHEMA TEXT DEFAULT NULL,
  CREATED_TIME   BIGINT NOT NULL,
  ORG_ID         VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (VERSION_ID),
  UNIQUE KEY uk_element_id_version (ORG_ID, ID, VERSION),                    -- C1: no duplicate versions per element
  UNIQUE KEY uk_element_name_ns_version (ORG_ID, NAME, NAMESPACE, VERSION),  -- C2: no duplicate name+ns at same version
  INDEX idx_id (ID),
  INDEX idx_name (NAME),
  INDEX idx_namespace (NAMESPACE),
  INDEX idx_type (TYPE),
  INDEX idx_org_id (ORG_ID)
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Properties for element versions (key/value pairs scoped to element version)
CREATE TABLE IF NOT EXISTS ELEMENT_PROPERTY (
  ELEMENT_VERSION_ID  CHAR(36) NOT NULL,
  ATT_KEY             VARCHAR(255) NOT NULL,
  ATT_VALUE           VARCHAR(1024) NOT NULL,
  ORG_ID              VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (ELEMENT_VERSION_ID, ATT_KEY),
  INDEX idx_att_key_element (ATT_KEY),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT FK_ELEMENT_PROPERTY
    FOREIGN KEY (ELEMENT_VERSION_ID)
    REFERENCES ELEMENT (VERSION_ID)
    ON DELETE CASCADE
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Purpose table — each row is a specific version of a purpose.
-- ID groups all versions of the same purpose (application-assigned UUID, never reused).
-- NAME+GROUP_ID uniqueness per org is enforced at the application layer;
-- C2 provides a partial DB-level guard at the version granularity.
CREATE TABLE IF NOT EXISTS PURPOSE (
  VERSION_ID     CHAR(36) NOT NULL,
  ID             CHAR(36) NOT NULL,
  NAME           VARCHAR(255) NOT NULL,
  GROUP_ID       VARCHAR(255) NOT NULL,
  VERSION        INT UNSIGNED NOT NULL,
  DISPLAY_NAME   VARCHAR(255) DEFAULT NULL,
  DESCRIPTION    VARCHAR(1024) DEFAULT NULL,
  CREATED_TIME   BIGINT NOT NULL,
  ORG_ID         VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (VERSION_ID),
  UNIQUE KEY uk_purpose_id_version (ORG_ID, ID, VERSION),                       -- C1: no duplicate versions per purpose
  UNIQUE KEY uk_purpose_name_group_version (ORG_ID, NAME, GROUP_ID, VERSION),   -- C2: no duplicate name+group at same version
  INDEX idx_id (ID),
  INDEX idx_purpose_name (NAME),
  INDEX idx_purpose_group_id (GROUP_ID),
  INDEX idx_purpose_org_id (ORG_ID),
  INDEX idx_purpose_org_group (ORG_ID, GROUP_ID)
) ENGINE=INNODB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

-- Properties for purpose versions (key/value pairs scoped to purpose version)
CREATE TABLE IF NOT EXISTS PURPOSE_PROPERTY (
  PURPOSE_VERSION_ID  CHAR(36) NOT NULL,
  ATT_KEY             VARCHAR(255) NOT NULL,
  ATT_VALUE           VARCHAR(1024) NOT NULL,
  ORG_ID              VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  PRIMARY KEY (PURPOSE_VERSION_ID, ATT_KEY),
  INDEX idx_att_key_purpose (ATT_KEY),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT FK_PURPOSE_PROPERTY
    FOREIGN KEY (PURPOSE_VERSION_ID)
    REFERENCES PURPOSE (VERSION_ID)
    ON DELETE CASCADE
) ENGINE=INNODB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

-- Maps element versions to purpose versions with mandatory flag (defines versioned purpose structure)
CREATE TABLE IF NOT EXISTS PURPOSE_ELEMENT_MAPPING (
  PURPOSE_VERSION_ID   CHAR(36) NOT NULL,
  ELEMENT_VERSION_ID   CHAR(36) NOT NULL,
  MANDATORY            BOOLEAN NOT NULL DEFAULT FALSE,
  ORG_ID               VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',

  PRIMARY KEY (PURPOSE_VERSION_ID, ELEMENT_VERSION_ID),
  INDEX idx_purpose_element_purpose_ver (PURPOSE_VERSION_ID),
  INDEX idx_purpose_element_element_ver (ELEMENT_VERSION_ID),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT fk_purpose_element_purpose_ver
    FOREIGN KEY (PURPOSE_VERSION_ID)
    REFERENCES PURPOSE (VERSION_ID)
    ON DELETE CASCADE,
  CONSTRAINT fk_purpose_element_element_ver
    FOREIGN KEY (ELEMENT_VERSION_ID)
    REFERENCES ELEMENT (VERSION_ID)
    ON DELETE RESTRICT
) ENGINE=INNODB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

-- Maps consents to the specific purpose version they were created against
CREATE TABLE IF NOT EXISTS PURPOSE_CONSENT_MAPPING (
  CONSENT_ID         CHAR(36) NOT NULL,
  PURPOSE_VERSION_ID CHAR(36) NOT NULL,
  ORG_ID             VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',

  PRIMARY KEY (CONSENT_ID, PURPOSE_VERSION_ID),
  INDEX idx_purpose_consent_consent (CONSENT_ID),
  INDEX idx_purpose_consent_purpose_ver (PURPOSE_VERSION_ID),
  INDEX idx_org_id (ORG_ID),
  CONSTRAINT fk_purpose_consent_consent
    FOREIGN KEY (CONSENT_ID)
    REFERENCES CONSENT (CONSENT_ID)
    ON DELETE CASCADE,
  CONSTRAINT fk_purpose_consent_purpose_ver
    FOREIGN KEY (PURPOSE_VERSION_ID)
    REFERENCES PURPOSE (VERSION_ID)
    ON DELETE RESTRICT
) ENGINE=INNODB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

-- Stores user approval status and value for each element version in a consent
CREATE TABLE IF NOT EXISTS CONSENT_ELEMENT_APPROVAL (
  CONSENT_ID          CHAR(36) NOT NULL,
  PURPOSE_VERSION_ID  CHAR(36) NOT NULL,
  ELEMENT_VERSION_ID  CHAR(36) NOT NULL,
  APPROVED            BOOLEAN NOT NULL DEFAULT FALSE,
  VALUE               BLOB DEFAULT NULL,  -- user-provided value for this element
  ORG_ID              VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  -- One approval per element version per purpose version per consent
  PRIMARY KEY (CONSENT_ID, PURPOSE_VERSION_ID, ELEMENT_VERSION_ID),
  INDEX idx_approval_consent (CONSENT_ID),
  INDEX idx_approval_purpose_ver (PURPOSE_VERSION_ID),
  INDEX idx_approval_element_ver (ELEMENT_VERSION_ID),
  INDEX idx_approval_status (APPROVED),
  INDEX idx_org_id (ORG_ID),
  -- Element version must belong to the purpose version
  CONSTRAINT fk_approval_purpose_element_ver
    FOREIGN KEY (PURPOSE_VERSION_ID, ELEMENT_VERSION_ID)
    REFERENCES PURPOSE_ELEMENT_MAPPING (PURPOSE_VERSION_ID, ELEMENT_VERSION_ID)
    ON DELETE RESTRICT,
  -- Purpose version must belong to the consent
  CONSTRAINT fk_approval_consent_purpose_ver
    FOREIGN KEY (CONSENT_ID, PURPOSE_VERSION_ID)
    REFERENCES PURPOSE_CONSENT_MAPPING (CONSENT_ID, PURPOSE_VERSION_ID)
    ON DELETE CASCADE
) ENGINE=INNODB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
