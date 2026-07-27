import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

const docs = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/docs' }),
  schema: z.object({
    title: z.string().min(2).max(70),
    description: z.string().min(70).max(180),
    order: z.number().int().positive(),
    updatedAt: z.coerce.date(),
    tags: z.array(z.string().min(1)).min(1),
    targetQuery: z.string().min(3),
    searchIntent: z.enum([
      'navigational',
      'problem',
      'solution',
      'how-to',
      'troubleshooting',
      'comparison',
      'evaluation',
    ]),
    draft: z.boolean(),
    noindex: z.boolean(),
  }),
});

export const collections = { docs };
