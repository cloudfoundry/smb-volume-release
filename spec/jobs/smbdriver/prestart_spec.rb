require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'pre-start' do
    let(:template) {job.template('bin/pre-start')}

    context 'when the smbdriver is enabled' do
      let(:manifest_properties) do
        {}
      end
      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("exit 0")
      end
    end

    context 'when the smbdriver is disabled' do
      let(:manifest_properties) do
        {
            "disable" => true,
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).not_to include("exit 0")
      end
    end
  end
end
