import { ogImagePath, product, siteUrl } from '../data/product';

export type SchemaNode = Record<string, unknown>;

export interface Breadcrumb {
  name: string;
  path: string;
}

const personRef = { '@id': `${product.siteUrl}/#maintainer` };
const websiteRef = { '@id': `${product.siteUrl}/#website` };
const softwareRef = { '@id': `${product.siteUrl}/#software` };

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
    primaryImageOfPage: {
      '@type': 'ImageObject',
      url: siteUrl(ogImagePath(path)),
      width: 1200,
      height: 630,
    },
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
  updatedAt,
  tags = [],
}: {
  path: string;
  title: string;
  description: string;
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
    dateModified:
      updatedAt instanceof Date ? updatedAt.toISOString() : new Date(updatedAt).toISOString(),
    image: {
      '@type': 'ImageObject',
      url: siteUrl(ogImagePath(path)),
      width: 1200,
      height: 630,
    },
    author: personRef,
    isPartOf: websiteRef,
    about: softwareRef,
    mainEntityOfPage: url,
    inLanguage: 'en',
    ...(tags.length > 0 ? { keywords: tags } : {}),
  };
}
