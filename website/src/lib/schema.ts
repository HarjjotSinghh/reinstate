import { ogImagePath, product, siteUrl } from '../data/product';

export type SchemaNode = Record<string, unknown>;

export interface Breadcrumb {
  name: string;
  path: string;
}

const personRef = { '@id': `${product.siteUrl}/#maintainer` };
const websiteRef = { '@id': `${product.siteUrl}/#website` };
const softwareRef = { '@id': `${product.siteUrl}/#software` };

function socialImageSchema(path: string, title: string): SchemaNode {
  return {
    '@type': 'ImageObject',
    url: siteUrl(ogImagePath(path)),
    width: 1200,
    height: 630,
    caption: `Branded Reinstate social card for ${title}`,
  };
}

export function homepageSchema(): SchemaNode[] {
  return [
    {
      '@type': 'Person',
      '@id': personRef['@id'],
      name: product.maintainer.name,
      url: product.maintainer.url,
      sameAs: [product.maintainer.githubUrl],
    },
    {
      '@type': 'WebSite',
      '@id': websiteRef['@id'],
      url: siteUrl('/'),
      name: product.name,
      description: product.shortDefinition,
      publisher: personRef,
      inLanguage: 'en',
    },
    {
      '@type': 'SoftwareApplication',
      '@id': softwareRef['@id'],
      name: product.name,
      url: siteUrl('/'),
      description: product.definition,
      applicationCategory: 'DeveloperApplication',
      applicationSubCategory: product.category,
      operatingSystem: [...product.supportedOperatingSystems],
      softwareVersion: product.currentRelease,
      isAccessibleForFree: true,
      image: siteUrl(product.defaultOgImage),
      offers: {
        '@type': 'Offer',
        price: '0',
        priceCurrency: 'USD',
      },
      downloadUrl: product.releasesUrl,
      softwareHelp: siteUrl('/docs'),
      author: personRef,
      license: product.licenseUrl,
    },
    {
      '@type': 'SoftwareSourceCode',
      '@id': `${product.siteUrl}/#source`,
      name: 'Reinstate source code',
      description: product.definition,
      codeRepository: product.repositoryUrl,
      programmingLanguage: product.programmingLanguage,
      runtimePlatform: [...product.supportedOperatingSystems],
      license: product.licenseUrl,
      author: personRef,
      targetProduct: softwareRef,
    },
  ];
}

export function breadcrumbSchema(items: Breadcrumb[]): SchemaNode {
  return {
    '@type': 'BreadcrumbList',
    itemListElement: items.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.name,
      item: siteUrl(item.path),
    })),
  };
}

export function webPageSchema({
  path,
  title,
  description,
  updatedAt,
}: {
  path: string;
  title: string;
  description: string;
  updatedAt?: Date | string;
}): SchemaNode {
  const url = siteUrl(path);
  return {
    '@type': 'WebPage',
    '@id': `${url}#webpage`,
    url,
    name: title,
    description,
    primaryImageOfPage: socialImageSchema(path, title),
    ...(updatedAt
      ? {
          dateModified:
            updatedAt instanceof Date
              ? updatedAt.toISOString()
              : new Date(updatedAt).toISOString(),
        }
      : {}),
    isPartOf: websiteRef,
    about: softwareRef,
    inLanguage: 'en',
  };
}

export function techArticleSchema({
  path,
  title,
  description,
  publishedAt,
  updatedAt,
  tags = [],
}: {
  path: string;
  title: string;
  description: string;
  publishedAt?: Date | string;
  updatedAt: Date | string;
  tags?: string[];
}): SchemaNode {
  const url = siteUrl(path);
  return {
    '@type': 'TechArticle',
    '@id': `${url}#article`,
    headline: title,
    description,
    url,
    ...(publishedAt
      ? {
          datePublished:
            publishedAt instanceof Date
              ? publishedAt.toISOString()
              : new Date(publishedAt).toISOString(),
        }
      : {}),
    dateModified:
      updatedAt instanceof Date ? updatedAt.toISOString() : new Date(updatedAt).toISOString(),
    image: socialImageSchema(path, title),
    author: personRef,
    isPartOf: websiteRef,
    about: softwareRef,
    mainEntityOfPage: url,
    inLanguage: 'en',
    ...(tags.length > 0 ? { keywords: tags } : {}),
  };
}

export function blogPostingSchema({
  path,
  title,
  description,
  publishedAt,
  updatedAt,
  tags = [],
}: {
  path: string;
  title: string;
  description: string;
  publishedAt: Date | string;
  updatedAt: Date | string;
  tags?: string[];
}): SchemaNode {
  const url = siteUrl(path);
  return {
    '@type': 'BlogPosting',
    '@id': `${url}#article`,
    headline: title,
    description,
    url,
    datePublished:
      publishedAt instanceof Date
        ? publishedAt.toISOString()
        : new Date(publishedAt).toISOString(),
    dateModified:
      updatedAt instanceof Date ? updatedAt.toISOString() : new Date(updatedAt).toISOString(),
    image: socialImageSchema(path, title),
    author: personRef,
    publisher: personRef,
    isPartOf: websiteRef,
    about: softwareRef,
    mainEntityOfPage: url,
    inLanguage: 'en',
    ...(tags.length > 0 ? { keywords: tags } : {}),
  };
}

export interface FaqEntry {
  question: string;
  answer: string;
}

export function faqPageSchema(path: string, entries: FaqEntry[]): SchemaNode {
  return {
    '@type': 'FAQPage',
    '@id': `${siteUrl(path)}#faq`,
    url: siteUrl(path),
    inLanguage: 'en',
    mainEntity: entries.map((entry) => ({
      '@type': 'Question',
      name: entry.question,
      acceptedAnswer: {
        '@type': 'Answer',
        text: entry.answer,
      },
    })),
  };
}
