import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

const searchIntent = z.enum([
  'navigational',
  'problem',
  'solution',
  'how-to',
  'troubleshooting',
  'comparison',
  'evaluation',
]);

const docs = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/docs' }),
  schema: z.object({
    title: z.string().min(2).max(70),
    description: z.string().min(70).max(180),
    order: z.number().int().positive(),
    author: z.string().min(2).max(80),
    status: z.enum(['current', 'planned', 'deprecated']),
    updatedAt: z.coerce.date(),
    tags: z.array(z.string().min(1)).min(1),
    targetQuery: z.string().min(3),
    searchIntent,
    draft: z.boolean(),
    noindex: z.boolean(),
  }),
});

const relatedLink = z.object({
  title: z.string().min(2).max(80),
  path: z.string().regex(/^\/[a-z0-9/-]*$/, 'Related links must use a site-relative path.'),
});

const editorialMetadata = z.object({
  title: z.string().min(10).max(70),
  description: z.string().min(70).max(180),
  answer: z.string().min(80).max(360),
  author: z.string().min(2).max(80),
  publishedAt: z.coerce.date(),
  updatedAt: z.coerce.date(),
  reviewedAt: z.coerce.date(),
  tags: z.array(z.string().min(2).max(60)).min(2).max(10),
  targetQuery: z.string().min(3).max(120),
  searchIntent,
  related: z.array(relatedLink).min(2).max(6),
  draft: z.boolean(),
  noindex: z.boolean(),
});

function validateEditorialDates(
  data: {
    publishedAt: Date;
    updatedAt: Date;
    reviewedAt: Date;
  },
  context: z.RefinementCtx,
) {
  if (data.updatedAt < data.publishedAt) {
    context.addIssue({
      code: 'custom',
      path: ['updatedAt'],
      message: 'updatedAt cannot be earlier than publishedAt.',
    });
  }

  if (data.reviewedAt < data.updatedAt) {
    context.addIssue({
      code: 'custom',
      path: ['reviewedAt'],
      message: 'reviewedAt cannot be earlier than updatedAt.',
    });
  }
}

const guides = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/guides' }),
  schema: editorialMetadata
    .extend({
      agent: z.enum(['claude-code', 'codex', 'general']),
      difficulty: z.enum(['beginner', 'intermediate', 'advanced']),
      estimatedMinutes: z.number().int().min(1).max(120),
      prerequisites: z.array(z.string().min(2).max(140)).min(1).max(10),
    })
    .superRefine(validateEditorialDates),
});

const blog = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/blog' }),
  schema: editorialMetadata
    .extend({
      category: z.enum(['engineering', 'product', 'security', 'open-source']),
      featured: z.boolean(),
    })
    .superRefine(validateEditorialDates),
});

export const collections = { docs, guides, blog };
