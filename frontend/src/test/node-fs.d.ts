// 前端测试运行在 Node 上，个别测试需要读取源码文本（例如断言 CSS 布局规则）。
// 项目没有引入 @types/node，这里只声明实际用到的那一个签名，避免为一条断言增加依赖。
declare module "node:fs" {
  export function readFileSync(path: URL | string, encoding: "utf8"): string
}
