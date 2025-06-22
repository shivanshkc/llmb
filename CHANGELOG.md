# [1.3.0](https://github.com/shivanshkc/llmb/compare/v1.2.0...v1.3.0) (2025-06-22)


### Bug Fixes

* **core:** json injection vuln and response body close ([494b1bc](https://github.com/shivanshkc/llmb/commit/494b1bc4e86d38550eee77a09eb9180038fbc656))
* **core:** ReadServerSentEvents correctly handles context expiry ([c002912](https://github.com/shivanshkc/llmb/commit/c002912a6f55cc541070cbcfb34bef877059dd8e))
* **core:** RetryClient context expiry fix ([2106448](https://github.com/shivanshkc/llmb/commit/2106448f89853d2ebdbe8cd781f2f1ad443cb6c6))
* **core:** TT timing fix, avoid redundant sorting ([e766433](https://github.com/shivanshkc/llmb/commit/e766433ed6672e987d3788830ea3cbbbd3880fab))
* **core:** unncessary sync.Once removed ([21f5fc7](https://github.com/shivanshkc/llmb/commit/21f5fc7b587d0681bbc667c8b9d26703ca9bf808))


### Features

* **core:** add the streams package ([f783e28](https://github.com/shivanshkc/llmb/commit/f783e287dbf94a0d379c66f09f8fd94b452f474e))
* **core:** chat command respects context expiry ([2f057f8](https://github.com/shivanshkc/llmb/commit/2f057f86fc3a65675ac3bef8506b79f46bf80b11))
* **core:** streams package is context aware ([b7fbb12](https://github.com/shivanshkc/llmb/commit/b7fbb1242c9197ec9560f316878f32fc1620fa01))

# [1.2.0](https://github.com/shivanshkc/llmb/compare/v1.1.0...v1.2.0) (2025-06-19)


### Features

* **core:** user can assume any role ([1bdf62a](https://github.com/shivanshkc/llmb/commit/1bdf62ad0d6a32074efd993af76394f1a5477e17))

# [1.1.0](https://github.com/shivanshkc/llmb/compare/v1.0.0...v1.1.0) (2025-06-19)


### Features

* **core:** add model flag to all commands ([b2996a5](https://github.com/shivanshkc/llmb/commit/b2996a5d044b015a852ced1801923f4252e78a2d))

# 1.0.0 (2025-06-19)


### Bug Fixes

* **core:** add event indexing ([321d6e8](https://github.com/shivanshkc/llmb/commit/321d6e8abc448aa8419da7bc38ac06255211694d))
* **docs:** add readme ([585e175](https://github.com/shivanshkc/llmb/commit/585e1757b9f0a55d1d606f772b99672279747091))


### Features

* **ci:** add ci-cd pipelines, update .gitignore, add LICENSE ([af76000](https://github.com/shivanshkc/llmb/commit/af7600043af6df5ce9125c7c43d5be81adc8de6d))
* **core:** add command options ([088c4dd](https://github.com/shivanshkc/llmb/commit/088c4dd831fe7a70f70c02d02e5bbb9a0026555a))
* **core:** add linter, gitignore, and LLM stream API abstraction ([cc37077](https://github.com/shivanshkc/llmb/commit/cc37077726b69e3333bd913828dd1dd763ceac08))
* **core:** add pretty table output ([3c0945d](https://github.com/shivanshkc/llmb/commit/3c0945d775dfe5e8d9b3827591c9197aaf732c75))
* **core:** add the debug flag for simple chat ([b893af7](https://github.com/shivanshkc/llmb/commit/b893af7443af65111a090dcaef15388c923e5fdf))
* **core:** bench command complete ([caa9760](https://github.com/shivanshkc/llmb/commit/caa9760f395ab27b3a4f7d38f47f80bcbcaf38a6))
* **core:** chat command complete ([e8eef14](https://github.com/shivanshkc/llmb/commit/e8eef145f80356bf864b5e5f81980a5b1977f733))
* **core:** ChatCompletionEvent implements the Event interface ([e09fae2](https://github.com/shivanshkc/llmb/commit/e09fae2f65d02a087965c2a78fea2c8d5e748c49))
* **core:** setup cobra commands ([4fe4204](https://github.com/shivanshkc/llmb/commit/4fe420413466af6ebbb1c7ba25b0e01778f9460e))
