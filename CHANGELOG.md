# Changelog

## [0.3.0](https://github.com/dortanes/ravenpass/compare/v0.2.0...v0.3.0) (2026-10-07)


### Features

* card autofill in the extension; storage screen for an unreadable vault ([#19](https://github.com/dortanes/ravenpass/issues/19)) ([7a62a86](https://github.com/dortanes/ravenpass/commit/7a62a86eb024fe8769b5d351d3695e58630c4482))
* trash, password merge, generator, breach checks ([#16](https://github.com/dortanes/ravenpass/issues/16)) ([decfb38](https://github.com/dortanes/ravenpass/commit/decfb3856a88da45e4b1ea6b7f88dac7933531c8))
* **ui:** ✨ add a Generator place with a full history of generated passwords ([#15](https://github.com/dortanes/ravenpass/issues/15)) ([d88b9c0](https://github.com/dortanes/ravenpass/commit/d88b9c095191ac6220e68a69102c3b1c1ef9765a))


### Bug Fixes

* keep setup on a failed way in; unsigned builds' keys ([#18](https://github.com/dortanes/ravenpass/issues/18)) ([c805582](https://github.com/dortanes/ravenpass/commit/c805582be69bff84f797c3ca590c420e1eabd9a5))


### Dependencies

* bump gradle-wrapper from 9.7.1 to 9.8.0 in /apps/mobile/build/android in the gradle group ([#8](https://github.com/dortanes/ravenpass/issues/8)) ([e9269a4](https://github.com/dortanes/ravenpass/commit/e9269a4f0764b043487de0d24e0e084f34ecdd7b))
* bump the go group across 2 directories with 2 updates ([#2](https://github.com/dortanes/ravenpass/issues/2)) ([9717e3d](https://github.com/dortanes/ravenpass/commit/9717e3dc7f08de84356dd017b5a3d492b2855ed4))
* bump the npm-production group across 1 directory with 12 updates ([#11](https://github.com/dortanes/ravenpass/issues/11)) ([ad5e5dc](https://github.com/dortanes/ravenpass/commit/ad5e5dce15b0fe6c007abdc5cd655f8bc62c9f45))

## [0.2.0](https://github.com/dortanes/ravenpass/compare/v0.1.1...v0.2.0) (2026-10-07)


### Features

* ✨ add item tags and QR code 2FA setups ([#12](https://github.com/dortanes/ravenpass/issues/12)) ([e21d528](https://github.com/dortanes/ravenpass/commit/e21d5287db2b4c6afc86473ff9665c17b7a551b7))
* **app:** ✨ duplicate items of any kind ([678d21a](https://github.com/dortanes/ravenpass/commit/678d21acb1f61d694101fa4a678cae8fe704af57))
* **app:** ✨ record the light, dark or system appearance ([3cb2f4a](https://github.com/dortanes/ravenpass/commit/3cb2f4a002a806031aa398ace44eba49febbcfc1))
* **desktop:** ✨ follow the chosen appearance on macOS ([418415f](https://github.com/dortanes/ravenpass/commit/418415f01d0229ec8e3533be1bc032ba1cd47ac9))
* **extension:** ✨ give the page menu a light frost ([cc66933](https://github.com/dortanes/ravenpass/commit/cc66933fb1a5bf652a27074b7d4dfa50491b82e0))
* **extension:** ✨ open field menus from context menu ([54ae069](https://github.com/dortanes/ravenpass/commit/54ae069f3e0d3685fd01bbb994f1f6b1ae158d63))
* **mobile:** ✨ follow the chosen appearance on Android ([ab49abe](https://github.com/dortanes/ravenpass/commit/ab49abe9770cc8d8da1cd9211058d984e94f9794))
* **ui:** ✨ add a light palette beside the dark one ([de7344f](https://github.com/dortanes/ravenpass/commit/de7344facb86a80e7777d6785732b195f058a4f0))
* **ui:** ✨ choose the appearance in settings ([2a9bc26](https://github.com/dortanes/ravenpass/commit/2a9bc26cf2183281d2a63c88786a3768d56c150a))
* **ui:** ✨ duplicate and delete from the item header ([3e0f5b7](https://github.com/dortanes/ravenpass/commit/3e0f5b7c085f90390d912ef61e3b77f52d161555))
* **ui:** ✨ warn on replaced keys, confirm by recovery key ([17089f9](https://github.com/dortanes/ravenpass/commit/17089f93b387b61064b401a45a006997cdc73028))
* **unlock:** ✨ ask for Touch ID before the panel ([1a3b512](https://github.com/dortanes/ravenpass/commit/1a3b512b1fef26276529987a5c00b8f7f7c628d3))
* **vault:** ✨ copy a stored scan to another document ([4407608](https://github.com/dortanes/ravenpass/commit/4407608684bac5379b87b1902ac4e8cbc93967d8))
* **vault:** ✨ name the key a session or envelope holds ([51de7f3](https://github.com/dortanes/ravenpass/commit/51de7f3154949a38816ed27e9ce2d5899d240c41))


### Bug Fixes

* 🐛 resolve interface and copy audit findings ([df903ab](https://github.com/dortanes/ravenpass/commit/df903ab136f27a071c273985b3e65665d26b51ef))
* **app:** 🐛 harden key changes, locks and following ([887289e](https://github.com/dortanes/ravenpass/commit/887289e378222d62c285d0dc6921e21aa64d2c2b))
* **extension:** 🐛 stack the locked menu vertically ([21c13f1](https://github.com/dortanes/ravenpass/commit/21c13f1da0f1b575be99befa9b0dee64e6253c22))
* **extension:** 🐛 take only the isolated script's port ([04d09ea](https://github.com/dortanes/ravenpass/commit/04d09ea3015e3adb0472ae30cf9d324094befcdf))
* **mobile:** 🐛 harden autofill, documents and the core ([aa1f340](https://github.com/dortanes/ravenpass/commit/aa1f340c4a56a476c34c1ace95cdec214d536d71))
* **mobile:** 🐛 hide the WebView's own scroll bar ([5c26f49](https://github.com/dortanes/ravenpass/commit/5c26f49bc025ca292b12df265b05722202f6b583))
* **ui:** 🐛 edge the PIN field in the light theme ([9986050](https://github.com/dortanes/ravenpass/commit/9986050962c535fc7955f665383c67e5c883bd8e))
* **ui:** 🐛 switch vaults without failures, unlock at once ([08716e7](https://github.com/dortanes/ravenpass/commit/08716e7aacec53cfd1c9337962af75216cd8ea04))
* **ui:** 🐛 wrap the confirmation card's buttons ([c3bbe58](https://github.com/dortanes/ravenpass/commit/c3bbe58873e50deb9637bf2bd0224524d892486f))

## [0.1.1](https://github.com/dortanes/ravenpass/compare/v0.1.0...v0.1.1) (2026-09-27)


### Bug Fixes

* **extension:** 🐛 use the Chrome Web Store extension ID ([b54af70](https://github.com/dortanes/ravenpass/commit/b54af70417954f2422ec21d9510c9a5733ba2c96))

## 0.1.0 (2026-09-27)


### Features

* ✨ initial public release ([e8f63dc](https://github.com/dortanes/ravenpass/commit/e8f63dc9c2f720a73d9d0e7ff6ee14a061692fa6))
