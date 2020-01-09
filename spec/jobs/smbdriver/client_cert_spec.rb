require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'client.crt' do
    let(:template) {job.template('config/certs/client.crt')}

    context 'when configured with a client_cert' do
      let(:manifest_properties) do
        {
            "tls" => {
                "client_cert" => "some-client-cert"
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("some-client-cert")
      end
    end
  end
end
