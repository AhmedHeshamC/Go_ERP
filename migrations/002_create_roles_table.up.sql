-- Migration: Create roles table
-- Created: ERPGo System
-- Description: Creates the roles table for role-based access control

-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    permissions TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for roles table
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles(created_at);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default roles
INSERT INTO roles (name, description, permissions) VALUES
('admin', 'System Administrator', ARRAY['users.create', 'users.read', 'users.update', 'users.delete', 'roles.create', 'roles.read', 'roles.update', 'roles.delete', 'system.admin']),
('teacher', 'Teacher with full access to educational resources', ARRAY['users.create', 'users.read', 'users.update', 'courses.create', 'courses.read', 'courses.update', 'courses.delete', 'assignments.create', 'assignments.read', 'assignments.update', 'assignments.delete', 'grades.create', 'grades.read', 'grades.update']),
('teaching_assistant', 'Teaching Assistant with limited access', ARRAY['users.read', 'courses.read', 'assignments.read', 'grades.read', 'grades.update']),
('student', 'Student with basic access', ARRAY['courses.read', 'assignments.read', 'grades.read', 'profile.read', 'profile.update'])
ON CONFLICT (name) DO NOTHING;