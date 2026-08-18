-- This creates the specific logical database for the IAM service
CREATE DATABASE iam_db;

-- Connect to the new database
\c iam_db;

-- Create the users table
CREATE TABLE users (
    id VARCHAR(50) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL
);