# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_VERSION = "2"

if Vagrant::VERSION == "1.8.7"
  path = `command -v curl`
  if path.include?("/opt/vagrant/embedded/bin/curl")
    puts "In Vagrant 1.8.7, curl is broken. Please use Vagrant 2.0.2 " \
         "or run 'sudo rm -f /opt/vagrant/embedded/bin/curl' to fix the " \
         "issue before provisioning. See " \
         "https://github.com/mitchellh/vagrant/issues/7997 " \
         "for reference."
    exit
  end
end

# Workaround: Vagrant removed the atlas.hashicorp.com to
# vagrantcloud.com redirect in February 2018. The value of
# DEFAULT_SERVER_URL in Vagrant versions less than 1.9.3 is
# atlas.hashicorp.com, which means that removal broke the fetching and
# updating of boxes (since the old URL doesn't work).  See
# https://github.com/hashicorp/vagrant/issues/9442
if Vagrant::DEFAULT_SERVER_URL == "atlas.hashicorp.com"
  Vagrant::DEFAULT_SERVER_URL.replace("https://vagrantcloud.com")
end

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure(VAGRANTFILE_VERSION) do |config|

  config.vm.hostname = "beast"

  # The default box for the machine is ubuntu/bionic64
  config.vm.box = "ubuntu/bionic64"

  # The Beast environment runs on 9991 on the guest.
  host_port = 5005
  http_proxy = https_proxy = no_proxy = nil
  host_ip_addr = "127.0.0.1"

  # System settings for the virtual machine.
  vm_num_cpus = "2"
  vm_memory = "2048"

  config.vm.network "forwarded_port", guest: 5005, host: host_port, host_ip: host_ip_addr
  config.vm.usable_port_range = 10000..20000

  # Mount beast repository on the vagrant box as a shared folder
  config.vm.synced_folder ".", "/home/vagrant/beast"

  # Provider-specific configuration so you can fine-tune various
  # backing providers for Vagrant. These expose provider-specific options.
  config.vm.provider "virtualbox" do |vb, override|
    override.vm.box = "ubuntu/bionic64"
    # Customize the amount of memory on the VM:
    vb.memory = vm_memory
  end

  config.vm.provider "hyperv" do |h, override|
    override.vm.box = "bento/ubuntu-18.04"
    h.memory = vm_memory
    h.maxmemory = vm_memory
    h.cpus = vm_num_cpus
  end

  config.vm.provider "parallels" do |prl, override|
    override.vm.box = "bento/ubuntu-18.04"
    override.vm.box_version = "202005.21.0"
    prl.memory = vm_memory
    prl.cpus = vm_num_cpus
  end

  config.vm.provision "docker"
  config.vm.provision "env", type: "shell", path: "scripts/installenv.sh", privileged: false
  config.vm.provision "setup", type: "shell", after: "env", path: "scripts/provision/setup.sh", privileged: false

end
