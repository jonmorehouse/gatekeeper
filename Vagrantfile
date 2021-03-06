# -*- mode: ruby -*-
# vi: set ft=ruby :

# A light weight vagrant VM for local development of gatekeeper and various
# plugins. By default, it uses ubuntu14.04 (trusty) and installs Docker for
# running various dependencies.
#
# NOTE: you might need to run `vagrant box add ubuntu/trusty64` before vagrant up
# NOTE: vagrant 1.8.4 is recommended, but for friendliness we aren't enforcing that.
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/trusty64"
  config.vm.synced_folder ".", "/gatekeeper"

  config.vm.provider 'virtualbox' do |v|
    v.name = 'gatekeeper'
    v.memory = 1024
    v.cpus = 2
  end

  # configure an IP address for easier ssh access, unless explicitly disabled
  if ENV["GATEKEEPER_NO_IP"].nil?
    ip = ENV["GATEKEEPER_PRIVATE_NETWORK_IP"] || "10.51.50.91"
    puts "creating private network with IP: #{ip}..."
    config.vm.network :private_network, ip: ip
  end

  def safe_read_file(path)
    if File.exists? path
      return File.read(path)
    end
  end

  # we copy the users public key into the virtual machine to make it friendlier to shell in and run commands
  ssh_public_key = safe_read_file(ENV["GATEKEEPER_SSH_PUBLIC_KEY"] || "#{ENV["HOME"]}/.ssh/id_rsa.pub")
  if ssh_public_key.nil? or ssh_public_key.empty?
    puts "gatekeeper development requires an ssh key to configure a vm..."
    puts "hint: you can set the `GATEKEEPER_SSH_PUBLIC_KEY` environment variable to specify a non default path..."
    exit 1
  end

  config.vm.provision "shell", inline: <<-SHELL
      apt-get update
      apt-get install -y curl make

      which go 2>&1 > /dev/null
      if [[ $? -ne 0 ]];then
        echo "installed go ..."
        curl https://storage.googleapis.com/golang/go1.7.0.linux-amd64.tar.gz | tar -C /usr/local -xzf -
        echo "installing go binaries into /usr/local/bin/ ..."
        ln -s /usr/local/go/bin/go /usr/local/bin/go
        ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
        ln -s /usr/local/go/bin/godoc /usr/local/bin/godoc
        chmod +x /usr/local/bin/*
      fi

      if [[ ! -d /gopath ]];then
        mkdir -p /gopath 
        mkdir -p /gopath/bin
        mkdir -p /gopath/src
      fi

      # update some base configuration needed
      echo -e "export GOPATH=/gopath" >> ~/.bash_profile
      echo -e "export GOPATH=/gopath" >> ~vagrant/.bash_profile
      echo -e "export PATH=$PATH:/gopath/bin" >> ~/.bash_profile
      echo -e "export PATH=$PATH:/gopath/bin" >> ~vagrant/.bash_profile

      which docker 2>&1 > /dev/null
      if [[ $? -ne 0 ]];then
        echo "installing docker ..."
        curl -s https://get.docker.com/ | sh
      fi

      echo "copying ssh public key into the vagrant user account ..."
      cat >> ~vagrant/.ssh/authorized_keys <<EOF
      #{ssh_public_key}
EOF

      echo "copying ssh public key into the root user account ..."
      mkdir -p /root/.ssh/
      cat >> /root/.ssh/authorized_keys <<EOF
      #{ssh_public_key}
EOF

      echo "symlinking /gatekeeper to root homedir ..."
      ln -sf /gatekeeper /root/gatekeeper
      echo "symlinking /gatekeeper to vagrant homedir ..."
      ln -sf /gatekeeper ~vagrant/gatekeeper
    SHELL
end
