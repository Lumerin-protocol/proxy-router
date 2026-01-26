import { readFileSync, writeFileSync } from "node:fs";
import { walk } from "./walk";

export function replaceInFiles(
  dir: string,
  replacements: Array<{ from: string | RegExp; to: string }>,
  extensions: string[] = [".sol"]
): number {
  let updatedFilesCount = 0;

  walk(
    dir,
    (filePath) => {
      let content = readFileSync(filePath, "utf-8");
      const originalContent = content;

      for (const { from, to } of replacements) {
        content = content.replace(from, to);
      }

      if (content !== originalContent) {
        writeFileSync(filePath, content);
        updatedFilesCount++;
      }
    },
    extensions
  );

  return updatedFilesCount;
}
