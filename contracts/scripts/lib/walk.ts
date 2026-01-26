import { readdirSync } from "node:fs";
import { join } from "node:path";

export function walk(
  dir: string,
  callback: (filePath: string) => void,
  extensions: string[] = [".sol"]
) {
  for (const file of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = join(dir, file.name);
    if (file.isDirectory()) {
      walk(fullPath, callback, extensions);
    }
    const ext = file.name.split(".").pop();
    if (ext && extensions.includes(`.${ext}`)) {
      callback(fullPath);
    }
  }
}
