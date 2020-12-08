Pod::Spec.new do |spec|
  spec.name         = 'Ggcl'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/gclchaineum/go-gclchaineum'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Gclchain Client'
  spec.source       = { :git => 'https://github.com/gclchaineum/go-gclchaineum.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Ggcl.framework'

	spec.prepare_command = <<-CMD
    curl https://ggclstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Ggcl.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
