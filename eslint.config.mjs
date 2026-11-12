// ESLint flat config（整个 frontend monorepo 共用一份）
//
// 目标：给 apps/demo-web 与 packages/* 的 TS/TSX 源码提供最小可用的静态检查，
// 规则集以「能报出真实问题、但不与现有代码风格打架」为准：
//   - js.configs.recommended          ESLint 官方基础规则
//   - typescript-eslint recommended   类型无关的 TS 规则（不做 type-aware，跑得快、不依赖 project）
//   - react flat recommended          已废弃写法的检查（含 jsx-runtime，无需手动 import React）
//   - react-hooks flat recommended    Hooks 依赖数组/调用位置等硬问题
//
// 说明：eslint-plugin-react 7.x 的 peer 范围是 eslint ^9.7，暂不支持 eslint 10，
// 因此本仓库固定 eslint 9.x；待 plugin 支持 10 再升级（见 frontend/package.json）。
import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import react from 'eslint-plugin-react'
import reactHooks from 'eslint-plugin-react-hooks'
import nxPlugin from '@nx/eslint-plugin'

const GLOBAL_IGNORES = [
  '**/dist/**',
  '**/node_modules/**',
  '**/coverage/**',
  '**/playwright-report/**',
  '**/test-results/**',
  '**/*.d.ts',
]

export default tseslint.config(
  { ignores: GLOBAL_IGNORES },
  // Nx 模块边界：注册 @nx 插件（flat/base 只注入插件），并在下方追加 enforce-module-boundaries 规则。
  ...nxPlugin.configs['flat/base'],
  {
    files: ['**/*.{ts,tsx,js,jsx}'],
    rules: {
      // A7：按 tags 约束依赖方向 —— type:lib 不得依赖 type:app；contracts 只依赖同类。
      '@nx/enforce-module-boundaries': [
        'error',
        {
          enforceBuildableLibDependency: false,
          allow: [],
          depConstraints: [
            // 应用（api/web/e2e）可依赖 公共库(lib) 与 契约(contracts)
            { sourceTag: 'type:app', onlyDependOnLibsWithTags: ['type:lib', 'type:contracts'] },
            // 公共库（libs-go / libs/ts/*）可依赖 其它库 与 契约，**不得依赖应用**
            { sourceTag: 'type:lib', onlyDependOnLibsWithTags: ['type:lib', 'type:contracts'] },
            // 契约平面只依赖契约，不被反向依赖到 app/lib 的具体实现
            { sourceTag: 'type:contracts', onlyDependOnLibsWithTags: ['type:contracts'] },
          ],
        },
      ],
    },
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      react,
      'react-hooks': reactHooks,
    },
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
    },
    settings: {
      react: { version: 'detect' },
    },
    rules: {
      ...react.configs.flat.recommended.rules,
      ...react.configs.flat['jsx-runtime'].rules,
      ...reactHooks.configs.flat.recommended.rules,
      // TS 已负责 props 类型，运行期 propTypes 不适用
      'react/prop-types': 'off',
      // 以下两条来自 react-hooks v7 的 React Compiler 规则集：它们针对的是
      // 既有页面的「挂载后 useEffect 里调 load() 拉数据」写法与 main.tsx 中
      // 按 IAM mode 动态构造 Provider 的写法，属于 pages/** 与 auth 既有实现
      // （不在本次构建任务范围内）。先降为 warning 保留信号，待页面统一改用
      // 数据请求库/loader 后再收紧为 error。
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/static-components': 'warn',
      // 与 tsconfig 的 noUnusedLocals 重复，且 TS 的检查更准确
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],
    },
  },
)
