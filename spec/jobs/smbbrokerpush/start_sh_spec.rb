require 'rspec'
require 'bosh/template/test'

describe 'smbbrokerpush job' do
  let(:release) { Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..')) }
  let(:job) { release.job('smbbrokerpush') }

  describe 'start.sh' do
    let(:template) { job.template('start.sh') }

    context 'when fully configured with all required credhub and log properties' do
      let(:manifest_properties) do
        {
            "credhub" => {
                "url" => "some-credhub-url",
                "uaa_client_id" => "client-id",
                "uaa_client_secret" => "client-secret",
                "store_id" => "some-store-id",
            },
            "log_level" => "some-log-level",
            "log_time_format" => "some-log-time-format",
        }
      end

      it 'successfully renders the script' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("bin/smbbroker --listenAddr=\"0.0.0.0:$PORT\"")
        expect(tpl_output).to include("--servicesConfig=\"./services.json\"")
        expect(tpl_output).to include("--credhubURL=\"some-credhub-url\"")
        expect(tpl_output).to include("--uaaClientID=\"client-id\"")
        expect(tpl_output).to include("--uaaClientSecret=\"client-secret\"")
        expect(tpl_output).to include("--storeID=\"some-store-id\"")
        expect(tpl_output).to include("--logLevel=\"some-log-level\"")
        expect(tpl_output).to include("--timeFormat=\"some-log-time-format\"")
      end
    end

    context 'when configured with all required credhub properties' do
      let(:manifest_properties) do
        {
            "credhub" => {
                "url" => "some-credhub-url",
                "uaa_client_id" => "some-uaa-client-id",
                "uaa_client_secret" => "some-uaa-client-secret",
                "store_id" => "some-store-id",
            }
        }
      end

      it 'includes the credhub flags in the script' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("--credhubURL=\"some-credhub-url\"")
        expect(tpl_output).to include("--uaaClientID=\"some-uaa-client-id\"")
        expect(tpl_output).to include("--uaaClientSecret=\"some-uaa-client-secret\"")
        expect(tpl_output).to include("--storeID=\"some-store-id\"")
      end
    end
  end
end
