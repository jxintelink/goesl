[![License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](https://github.com/jxintelink/goesl/blob/master/LICENSE)
[![Build Status](https://travis-ci.org/0x19/goesl.svg)](https://travis-ci.org/0x19/goesl)
[![Go 1.3 Ready](https://img.shields.io/badge/Go%201.3-Ready-green.svg?style=flat)]()
[![Go 1.4 Ready](https://img.shields.io/badge/Go%201.4-Ready-green.svg?style=flat)]()
[![Go 1.5 Ready](https://img.shields.io/badge/Go%201.5-Ready-green.svg?style=flat)]()

## FreeSWITCH Event Socket Library Wrapper for Go

GoESL is a small wrapper around [FreeSWITCH](https://freeswitch.org/) [Event Socket Library](https://wiki.freeswitch.org/wiki/Event_Socket_Library) written in [Go](http://golang.org).

Point of this library is to fully implement FreeSWITCH ESL and bring outbound server as inbound client to you, fellow Go developer :)

### Upstream / 源项目与原作者

本项目代码源自 **Nevio Vesic** 的开源项目 **[GoESL](https://github.com/0x19/goesl)**（GitHub：`github.com/0x19/goesl`）。原作者以 **MIT License** 发布，本仓库在**保留原有版权声明与许可条款**的前提下进行维护与扩展（例如 Go modules、Inbound 断线自动重连、`Client.Handle` 默认行为等）。若使用或再分发，请继续遵守根目录 [LICENSE](LICENSE) 中的要求。

**This fork** is based on **GoESL** by **Nevio Vesic** at [github.com/0x19/goesl](https://github.com/0x19/goesl). The original work is licensed under the **MIT License**; this repository carries forward the same license and copyright notice. Thanks to the original author for sharing the code.

- 上游仓库 / Upstream: <https://github.com/0x19/goesl>
- 本仓库 / This fork: <https://github.com/jxintelink/goesl>
- 模块路径 / Go module: `github.com/jxintelink/goesl`

### WARNING

Before using this code, please read following discussion: https://github.com/0x19/goesl/discussions/35

In short, I don't have time right now contributing to this project. 


### TODO?

You can find what still needs to be done at [GoESL TODO](https://github.com/0x19/goesl/blob/master/TODO.md)


### Examples

There are few different types of examples in this repository under [`examples/`](examples/). The original project also maintained examples at [0x19/goesl examples](https://github.com/0x19/goesl/tree/master/examples).

Feel free to suggest more examples :)


### Contributions / Issues?

Please reach me over nevio.vesic@gmail.com or visit my [website](http://www.neviovesic.com/) or submit new [issue](https://github.com/0x19/goesl/issues/new). I'd prefer tho if you would submit [issue](https://github.com/0x19/goesl/issues/new).


### License

The MIT License (MIT)

Copyright (c) 2015 Nevio Vesic

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
