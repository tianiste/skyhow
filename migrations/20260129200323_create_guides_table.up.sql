create table if not exists public.guides (
  id uuid primary key default gen_random_uuid(),

  creator_id uuid not null
    references public.users(id)
    on delete cascade,

  title text not null,
  content text not null, 

  status text not null default 'draft'
    check (status in ('draft', 'published')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_guides_creator_id on public.guides(creator_id);
create index if not exists idx_guides_status on public.guides(status);


