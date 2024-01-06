# silly-torrent-client

学习 GO，练手用。

ref：
1. [project-based-learning](https://github.com/practical-tutorials/project-based-learning?tab=readme-ov-file#go)
1. [BEP](https://www.bittorrent.org/beps/bep_0003.html) // 总感觉 BEP 的描述不甚清晰，后续可以探索下是否有更详细的 specification
1. https://www.morehawes.ca/old-guides/the-bittorrent-protocol
1. https://wiki.theory.org/BitTorrentSpecification

用本 repo 代码和其它 torrent client 分别下载 debian-12.4.0-amd64-netinst.iso.torrent，验证 shasum 相同以确定正确性。

感想是
1. GO 真的很简陋，但同时又在某些奇怪的地方很方便（比如无论指针还是非指针作为方法的接收者，调用时会自动转换；直接用 . 来 access 属性和方法而不区分指针和非指针也是），可能工程就是妥协吧
2. goroutine/channel 的组合，确实很有特色（但个人总觉得这个 goroutine 会导致 debug 困难，一个 channel stuck 的话，如果多个 goroutine 都涉及到这个 channel，就感觉变得一团乱麻了）
3. 以及作为我实际写过玩具代码的第一门编译型语言，还是挺新奇的，至少不用像 python 一样为环境管理而头疼
4. 容易标准化，比如 gofmt，以及 go mod 统一的项目管理，在看其他开源代码的时候，不会因为项目管理方面的 preference 给读代码带来太多的困难
5. if err != nil 确实很折磨。。我还是喜欢 try catch 这样的