-- Operators (server 0.2.0).
--
-- An operator may run the dashboard's operator screens: accounts, device
-- tokens, teams, server info. operatorAt NULL means a regular account, which
-- is what every existing account becomes; the first operator is named with
-- `croffle-server user operator grant <user>` (or `user create --operator`).
--
-- Being an operator changes nothing about sync: an operator's devices see the
-- same rows as anyone else's.
ALTER TABLE account ADD COLUMN operatorAt INTEGER;
