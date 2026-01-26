
create table if not exists public.sessions (
  id uuid primary key default gen_random_uuid(),  
  user_id uuid not null references public.users(id) on delete cascade,

  created_at timestamptz not null default now(),
  expires_at timestamptz not null
);

create index if not exists idx_sessions_user_id on public.sessions(user_id);
create index if not exists idx_sessions_expires_at on public.sessions(expires_at);

