import fs from "fs";
import path from "path";
import matter from "gray-matter";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import rehypeStringify from "rehype-stringify";
import { unified } from "unified";
import { Node, Parent } from "unist";
import { visit } from "unist-util-visit";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeSlug from "rehype-slug";

export async function processMarkdown(
  filePath: string,
  linksBasePath: string = ""
): Promise<{ contentHtml: string; toc: TOCItem[] }> {
  const fileContents = fs.readFileSync(filePath, "utf8");
  const { content } = matter(fileContents);
  let toc: TOCItem[] = [];

  const processor = unified()
    .use(remarkParse)
    .use(() => (tree) => {
      toc = extractTOC()(tree);
    })
    .use(remarkRehype)
    .use(rehypeSlug)
    //.use(rehypeAutolinkHeadings)
    .use(() => {
      return (tree: Node) => {
        visit(tree, "element", (node: any) => {
          if (node.tagName === "a" && node.properties && node.properties.href) {
            const href: string = node.properties.href;
            if (!href.startsWith("http") && !href.startsWith("#")) {
              // if it ends in .md remove it, and if it ends in index.md we remove that fully
              node.properties.href = `${linksBasePath}/${href.replace(
                /\/index\.md$/,
                ""
              ).replace(/\.md$/, "")}`;
            }
          }
        });
      };
    })
    .use(rehypeStringify);

  const processedContent = await processor.process(content);

  return {
    contentHtml: processedContent.toString(),
    toc,
  };
}

export function getDocPaths(docCategory: string): string[] {
  const docsDirectory = path.join(process.cwd(), "docs", docCategory);
  return fs.readdirSync(docsDirectory);
}

interface TOCItem {
  value: string;
  id: string;
  depth: number;
}

export async function getDocData(
  category: string,
  slug: string,
  linksBasePath: string = ""
): Promise<{
  slug: string;
  contentHtml: string;
  toc: TOCItem[];
  [key: string]: any;
}> {
  var fullPath = path.join(process.cwd(), "docs", category, `${slug}.md`);
  if (category === "") {
    fullPath = path.join(process.cwd(), "docs", `${slug}.md`);
  }

  const { contentHtml, toc } = await processMarkdown(fullPath, linksBasePath);

  return {
    slug,
    contentHtml,
    toc,
  };
}

// This is a simplified example to extract headings and create a TOC
function extractTOC(): (tree: Node) => TOCItem[] {
  let toc: TOCItem[] = [];
  return (tree) => {
    visit(tree, "heading", (node: any) => {
      const text = node.children
        .filter((n: any) => n.type === "text")
        .map((n: any) => n.value)
        .join("");
      const id = text
        .toLowerCase()
        .replace(/\s+/g, "-")
        .replace(/[^\w-]+/g, "");
      toc.push({ value: text, id, depth: node.depth });
    });
    return toc;
  };
}
