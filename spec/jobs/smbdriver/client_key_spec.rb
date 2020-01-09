require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'client.key' do
    let(:template) {job.template('config/certs/client.key')}

    context 'when configured with a client key' do
      let(:manifest_properties) do
        {
            "tls" => {
                "client_key" => "some-client_key"
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("some-client_key")
      end
    end
  end
end
