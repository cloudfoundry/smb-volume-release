require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'server.key' do
    let(:template) {job.template('config/certs/server.key')}

    context 'when configured with a server key' do
      let(:manifest_properties) do
        {
            "tls" => {
                "server_key" => "some-server-key"
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("some-server-key")
      end
    end
  end
end
