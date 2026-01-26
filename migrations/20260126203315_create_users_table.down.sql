drop trigger if exists trg_users_updated_at on public.users;
drop function if exists public.set_users_updated_at();

drop index if exists idx_users_created_at;
drop index if exists idx_users_role;
drop index if exists ux_users_email_not_null;

drop table if exists public.users;

