{ buildGo125Module
, self
, lib
}:
buildGo125Module {
  pname = "ts-fileserver";
  version = "${builtins.readFile ./version.txt}-${self.shortRev or self.dirtyShortRev or "rev"}";

  src = ./.;

  env.CGO_ENABLED = 0;

  # vendorHash = "sha256-EkqIQSaD9sL6Y/6K0lpIo7maO4ZWhgd3d0QtmuAQPRU=";
  vendorHash = "sha256-EkqIQSaD9sL6Y/6K0lpIo7maO4ZWhgd3d0QtmuAQPRU=";

  postConfigure = ''
    # chmod -R +w vendor/gvisor.dev/gvisor #/pkg/refs/refs_template.go
    # rm vendor/gvisor.dev/gvisor/pkg/refs/refs_template.go
    # substituteInPlace vendor/gvisor.dev/gvisor/pkg/refs/refs_template.go \
    #   --replace refs_template refs
  '';

  meta.mainProgram = "ts-proxyd";
}
