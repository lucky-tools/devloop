class Devloop < Formula
  desc "A tool that facilitates continuous development for Kubernetes applications."
  head "https://github.com/lucky-tools/devloop.git"
  url "https://github.com/lucky-tools/devloop.git"

  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    (buildpath/"src/github.com/lucky-tools").mkpath

    ln_s buildpath, buildpath/"src/github.com/lucky-tools/devloop"
    system "make"
    bin.install "out/devloop"
  end

  test do
    system "#{bin}devloop", "--version"
  end
end
