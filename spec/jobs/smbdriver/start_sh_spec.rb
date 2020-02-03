require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'start.sh' do
    let(:template) {job.template('bin/smbdriver_ctl')}

    context 'when fully configured' do
      let(:manifest_properties) do
        {
            "listen_port" => "1111",
            "debug_addr" => "2222",
            "driver_path" => "/some/driver/path",
            "cell_mount_path" => "/some/cell/mount/path",
            "log_level" => "some-log-level",
            "log_time_format" => "some-log-level-format",
            "allowed_in_mount" => "some,options",
            "default_in_mount" => "some,default,options",
            "enable_unique_volume_ids" => true,
            "tls" => {
                "ca_cert" => "some-ca-cert"
            },
            "ssl" => {
                "insecure_skip_verify" => true
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("--listenPort=1111")
        expect(tpl_output).to include("--debugAddr=\"2222\"")
        expect(tpl_output).to include("--driversPath=\"/some/driver/path\"")
        expect(tpl_output).to include("--mountDir=\"/some/cell/mount/path\"")
        expect(tpl_output).to include("--logLevel=\"some-log-level\"")
        expect(tpl_output).to include("--timeFormat=\"some-log-level-format\"")
        expect(tpl_output).to include("--requireSSL")
        expect(tpl_output).to include("/server.crt")
        expect(tpl_output).to include("/server.key")
        expect(tpl_output).to include("/ca.crt")
        expect(tpl_output).to include("/client.crt")
        expect(tpl_output).to include("/client.key")
        expect(tpl_output).to include("--insecureSkipVerify")
      end
    end

    context 'when not configured with tls' do
      let(:manifest_properties) do
        {
            "listen_port" => "1111",
            "debug_addr" => "2222",
            "driver_path" => "/some/driver/path",
            "cell_mount_path" => "/some/cell/mount/path",
            "log_level" => "some-log-level",
            "log_time_format" => "some-log-level-format",
            "allowed_in_mount" => "some,options",
            "default_in_mount" => "some,default,options",
            "enable_unique_volume_ids" => true,
            "ssl" => {
                "insecure_skip_verify" => true
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).not_to include("--requireSSL")
        expect(tpl_output).not_to include("/server.crt")
        expect(tpl_output).not_to include("/server.key")
        expect(tpl_output).not_to include("/ca.crt")
        expect(tpl_output).not_to include("/client.crt")
        expect(tpl_output).not_to include("/client.key")
      end
    end

    context 'when configured with tls and skip ssl verification' do
      let(:manifest_properties) do
        {
            "tls" => {
                "ca_cert" => "some-ca-cert"
            },
            "ssl" => {
                "insecure_skip_verify" => false
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("--requireSSL")
        expect(tpl_output).to include("/server.crt")
        expect(tpl_output).to include("/server.key")
        expect(tpl_output).to include("/ca.crt")
        expect(tpl_output).to include("/client.crt")
        expect(tpl_output).to include("/client.key")
        expect(tpl_output).not_to include("--insecureSkipVerify")
      end
    end
  end
end
