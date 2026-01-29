create table if not exists public.guide_tags (
  guide_id uuid not null
    references public.guides(id)
    on delete cascade,

  tag_id uuid not null
    references public.tags(id)
    on delete cascade,

  primary key (guide_id, tag_id)
);

create index if not exists idx_guide_tags_tag_id on public.guide_tags(tag_id);

