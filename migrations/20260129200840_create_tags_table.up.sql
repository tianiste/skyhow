create table if not exists public.tags (
  id uuid primary key default gen_random_uuid(),
  name text not null unique
);

