/* Navigation tree for go-z docs */
window.DOCS_NAV = [
  {
    title: "Start here",
    items: [
      { title: "Introduction", path: "/" },
      { title: "Installation", path: "/guide/installation" },
      { title: "Quickstart", path: "/guide/quickstart" },
      { title: "Why go-z?", path: "/guide/why" },
      { title: "Comparison", path: "/guide/comparison" },
    ],
  },
  {
    title: "Core concepts",
    items: [
      { title: "Schemas & parsing", path: "/guide/parsing" },
      { title: "Issues & errors", path: "/guide/errors" },
      { title: "Error maps & locales", path: "/guide/error-maps" },
      { title: "Checks & refinements", path: "/guide/checks" },
      { title: "Missing vs nil", path: "/guide/missing-nil" },
      { title: "Immutability & concurrency", path: "/guide/concurrency" },
    ],
  },
  {
    title: "Primitives",
    items: [
      { title: "String", path: "/api/string" },
      { title: "String formats", path: "/api/string-formats" },
      { title: "Number & Int", path: "/api/number" },
      { title: "BigInt", path: "/api/bigint" },
      { title: "Bool", path: "/api/bool" },
      { title: "Time", path: "/api/time" },
      { title: "Literal & Enum", path: "/api/literal-enum" },
      { title: "Special types", path: "/api/special" },
      { title: "Coercion", path: "/api/coerce" },
    ],
  },
  {
    title: "Collections",
    items: [
      { title: "Object", path: "/api/object" },
      { title: "Array", path: "/api/array" },
      { title: "Tuple", path: "/api/tuple" },
      { title: "Record", path: "/api/record" },
      { title: "Map & Set", path: "/api/map-set" },
    ],
  },
  {
    title: "Composition",
    items: [
      { title: "Optional & Nullable", path: "/api/optional" },
      { title: "Default, Prefault & Catch", path: "/api/defaults" },
      { title: "Union", path: "/api/union" },
      { title: "Xor (exclusive union)", path: "/api/xor" },
      { title: "Discriminated union", path: "/api/discriminated-union" },
      { title: "Intersection", path: "/api/intersection" },
      { title: "Lazy & recursive", path: "/api/lazy" },
      { title: "Pipe & Transform", path: "/api/pipe-transform" },
      { title: "Codec (encode/decode)", path: "/api/codec" },
      { title: "Template literal", path: "/api/template-literal" },
      { title: "Refine & Custom", path: "/api/refine" },
    ],
  },
  {
    title: "Interop",
    items: [
      { title: "JSON Schema export", path: "/api/json-schema" },
      { title: "Registries & metadata", path: "/api/registry" },
    ],
  },
  {
    title: "Integrations",
    items: [
      { title: "Gin (zgin)", path: "/integrations/gin" },
      { title: "Struct binding", path: "/integrations/tostruct" },
      { title: "HTTP error shapes", path: "/integrations/http-errors" },
    ],
  },
  {
    title: "Performance",
    items: [
      { title: "Performance model", path: "/guide/performance" },
      { title: "Parallel validation", path: "/guide/parallel" },
      { title: "Benchmarks", path: "/guide/benchmarks" },
    ],
  },
  {
    title: "Reference",
    items: [
      { title: "API cheat sheet", path: "/reference/cheatsheet" },
      { title: "Issue codes", path: "/reference/issue-codes" },
      { title: "Migration from Zod", path: "/reference/migration" },
      { title: "Cookbook", path: "/reference/cookbook" },
      { title: "FAQ", path: "/reference/faq" },
    ],
  },
];

window.DOCS_PAGES = (function () {
  const pages = [];
  for (const group of window.DOCS_NAV) {
    for (const item of group.items) {
      pages.push({ ...item, group: group.title });
    }
  }
  return pages;
})();
