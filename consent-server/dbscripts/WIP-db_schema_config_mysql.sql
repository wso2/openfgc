-- Configuration Management API Database Schema
-- Version: 1.0.0
-- Description: Schema for configuration resource management system with attributes
-- Compatible with: MySQL 8.0+

-- Drop tables if they exist (for clean reinstall)
DROP TABLE IF EXISTS CONFIG_ATTRIBUTE;
DROP TABLE IF EXISTS CONFIG_RESOURCE;

-- =====================================================================
-- Main configuration resource table
-- Stores configuration metadata for resources
-- =====================================================================
CREATE TABLE IF NOT EXISTS CONFIG_RESOURCE (
  RESOURCE_ID           VARCHAR(255) NOT NULL,
  ORG_ID                VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  RESOURCE_NAME         VARCHAR(255) NOT NULL,
  CREATED_TIME          BIGINT NOT NULL,
  LAST_MODIFIED         BIGINT NOT NULL,
  HAS_ATTRIBUTE         BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (RESOURCE_ID, ORG_ID),
  UNIQUE KEY UK_NAME_ORG (RESOURCE_NAME, ORG_ID),
  INDEX idx_resource_name (RESOURCE_NAME),
  INDEX idx_org_id (ORG_ID),
  INDEX idx_created_time (CREATED_TIME),
  INDEX idx_last_modified (LAST_MODIFIED)
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci

-- =====================================================================
-- Configuration attributes table
-- Stores key-value pairs associated with configuration resources
-- =====================================================================
CREATE TABLE IF NOT EXISTS CONFIG_ATTRIBUTE (
  ATTRIBUTE_ID          VARCHAR(255) NOT NULL,
  RESOURCE_ID           VARCHAR(255) NOT NULL,
  ORG_ID                VARCHAR(255) NOT NULL DEFAULT 'DEFAULT_ORG',
  ATTR_KEY              VARCHAR(255) NOT NULL,
  ATTR_VALUE            TEXT DEFAULT NULL,
  PRIMARY KEY (ATTRIBUTE_ID, ORG_ID),
  UNIQUE KEY UK_RESOURCE_KEY (RESOURCE_ID, ATTR_KEY, ORG_ID),
  INDEX idx_resource_id (RESOURCE_ID),
  INDEX idx_attr_key (ATTR_KEY),
  CONSTRAINT FK_CONFIG_ATTRIBUTE_RESOURCE
    FOREIGN KEY (RESOURCE_ID, ORG_ID)
    REFERENCES CONFIG_RESOURCE (RESOURCE_ID, ORG_ID)
    ON DELETE CASCADE
    ON UPDATE CASCADE
) ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci