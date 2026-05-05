-- Revocaciones antiguas guardaban el id de usuario como token_jti, bloqueando
-- cualquier refresh del mismo usuario. Los jti nuevos son hex de 32 caracteres.
DELETE FROM revoked_tokens
WHERE token_jti ~ '^[0-9]+$';
