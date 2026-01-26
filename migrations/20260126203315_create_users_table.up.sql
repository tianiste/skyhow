create extension if not exists "pgcrypto";

create table if not exists public.users (
  id uuid primary key default gen_random_uuid(),
  display_name text not null,
  avatar_url text null,
  email text null,
  email_verified boolean not null default false,
  role text not null default 'user',
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint users_role_check check (role in ('user', 'contributor', 'editor', 'admin'))
);

create unique index if not exists ux_users_email_not_null
  on public.users (lower(email))
  where email is not null;

create index if not exists idx_users_role on public.users(role);
create index if not exists idx_users_created_at on public.users(created_at);

create or replace function public.set_users_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

drop trigger if exists trg_users_updated_at on public.users;
create trigger trg_users_updated_at
before update on public.users
for each row
execute function public.set_users_updated_at();

