-- 000001_init_schema.down.sql
-- Revert Beast CTF platform database schema in reverse dependency order.

DROP TABLE IF EXISTS audit_events CASCADE;
DROP TABLE IF EXISTS auth_sessions CASCADE;
DROP TABLE IF EXISTS competition_config CASCADE;
DROP TABLE IF EXISTS user_notification_reads CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS solves CASCADE;
DROP TABLE IF EXISTS submissions CASCADE;
DROP TABLE IF EXISTS deployment_endpoints CASCADE;

DROP INDEX IF EXISTS uq_active_shared_challenge_deployment;
DROP INDEX IF EXISTS uq_active_team_challenge_deployment;
DROP FUNCTION IF EXISTS get_user_team_id(UUID);

DROP TABLE IF EXISTS challenge_deployments CASCADE;
DROP TABLE IF EXISTS challenge_instance_config CASCADE;
DROP TABLE IF EXISTS challenge_runtime_config CASCADE;
DROP TABLE IF EXISTS challenge_flags CASCADE;
DROP TABLE IF EXISTS challenge_files CASCADE;
DROP TABLE IF EXISTS user_hints CASCADE;
DROP TABLE IF EXISTS challenge_hints CASCADE;
DROP TABLE IF EXISTS challenge_prerequisites CASCADE;
DROP TABLE IF EXISTS challenge_tags CASCADE;
DROP TABLE IF EXISTS tags CASCADE;
DROP TABLE IF EXISTS challenges CASCADE;
DROP TABLE IF EXISTS auth_otps CASCADE;

ALTER TABLE IF EXISTS teams DROP CONSTRAINT IF EXISTS fk_teams_leader;
ALTER TABLE IF EXISTS teams DROP CONSTRAINT IF EXISTS fk_teams_creator;

DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
